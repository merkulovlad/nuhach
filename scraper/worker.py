import asyncio
import logging
import os
import time
from datetime import datetime, timedelta, timezone

import psycopg

from scraper.commentary import refine_comment
from scraper.matching import assess_price_risk, deduplicate_offers, enrich_and_filter
from scraper.models import Offer, PerfumeTarget
from scraper.openrouter_search import search_offers as openrouter_search_offers
from scraper.stores.registry import build_stores


logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)


class Worker:
    def __init__(self) -> None:
        self.database_url = os.environ["DATABASE_URL"]
        self.poll_seconds = float(os.getenv("SCRAPER_POLL_SECONDS", "3"))
        self.ttl_hours = int(os.getenv("OFFER_CACHE_TTL_HOURS", "8"))
        enabled = {
            value.strip()
            for value in os.getenv(
                "SCRAPER_STORES", "Ozon,ЛЭТУАЛЬ,Золотое Яблоко,Рандеву"
            ).split(",")
            if value.strip()
        }
        self.stores = build_stores(enabled)

    def run(self) -> None:
        logger.info("Offer worker started with %d configured stores", len(self.stores))
        while True:
            try:
                job = self.claim_job()
                if job is None:
                    time.sleep(self.poll_seconds)
                    continue
                asyncio.run(self.process(*job))
            except KeyboardInterrupt:
                return
            except Exception:
                logger.exception("Worker loop failed")
                time.sleep(self.poll_seconds)

    def claim_job(self) -> tuple[int, PerfumeTarget] | None:
        query = """
            WITH next_job AS (
                SELECT id
                FROM offer_search_jobs
                WHERE status = 'queued'
                ORDER BY requested_at
                FOR UPDATE SKIP LOCKED
                LIMIT 1
            )
            UPDATE offer_search_jobs j
            SET status = 'running', started_at = NOW(), updated_at = NOW(), error = NULL
            FROM next_job
            WHERE j.id = next_job.id
            RETURNING j.id, j.perfume_id
        """
        with psycopg.connect(self.database_url) as conn:
            row = conn.execute(query).fetchone()
            if row is None:
                return None
            job_id, perfume_id = row
            target_row = conn.execute(
                """
                SELECT p.id, COALESCE(b.name, ''), p.perfume_name
                FROM perfumes_normalized p
                LEFT JOIN brands b ON b.id = p.brand_id
                WHERE p.id = %s
                """,
                (perfume_id,),
            ).fetchone()
            if target_row is None:
                self.fail_job(conn, job_id, "perfume not found")
                return None
            return job_id, PerfumeTarget(*target_row)

    async def process(self, job_id: int, target: PerfumeTarget) -> None:
        try:
            if not self.stores:
                raise RuntimeError("no stores configured in SCRAPER_STORES")
            raw_offers: list[Offer] = []
            store_errors: list[str] = []
            results = await asyncio.gather(
                *(store.search(target) for store in self.stores),
                return_exceptions=True,
            )
            for store, result in zip(self.stores, results):
                if isinstance(result, Exception):
                    logger.warning("Store %s failed: %s", store.name, result)
                    store_errors.append(f"{store.name}: {result}")
                    continue
                raw_offers.extend(result)

            # OpenRouter complements direct adapters with per-store, domain-restricted
            # discovery. Direct adapters remain fail-closed on robots.txt.
            raw_offers.extend(await openrouter_search_offers(target))
            if not raw_offers and len(store_errors) == len(self.stores):
                raise RuntimeError("; ".join(store_errors))

            offers = enrich_and_filter(target, raw_offers)
            offers = deduplicate_offers(offers)
            assess_price_risk(offers)
            for offer in offers:
                offer.comment = await refine_comment(target, offer)
                if offer.is_catalog_link:
                    catalog_note = (
                        "Ссылка ведёт на каталог магазина — найдите товар по указанному названию."
                    )
                    offer.comment = f"{offer.comment} {catalog_note}".strip()
            self.save(job_id, target.id, offers)
            logger.info("Job %d completed with %d offers", job_id, len(offers))
        except Exception as exc:
            logger.exception("Job %d failed", job_id)
            with psycopg.connect(self.database_url) as conn:
                self.fail_job(conn, job_id, str(exc))

    def save(self, job_id: int, perfume_id: int, offers: list[Offer]) -> None:
        now = datetime.now(timezone.utc)
        expires_at = now + timedelta(hours=self.ttl_hours)
        with psycopg.connect(self.database_url) as conn:
            with conn.transaction():
                conn.execute("DELETE FROM store_offers WHERE perfume_id = %s", (perfume_id,))
                for offer in offers:
                    conn.execute(
                        """
                        INSERT INTO store_offers (
                            perfume_id, search_job_id, store, seller, title, price, old_price,
                            currency, volume_ml, concentration, product_type, in_stock, url,
                            rating, reviews_count, match_confidence, risk_level, risk_score,
                            comment, checked_at, expires_at
                        ) VALUES (
                            %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                            %s, %s, %s, %s, %s, %s, %s, %s
                        )
                        """,
                        (
                            perfume_id, job_id, offer.store, offer.seller, offer.title,
                            offer.price, offer.old_price, offer.currency, offer.volume_ml,
                            offer.concentration, offer.product_type, offer.in_stock, offer.url,
                            offer.rating, offer.reviews_count, offer.match_confidence,
                            offer.risk_level, offer.risk_score, offer.comment, now, expires_at,
                        ),
                    )
                conn.execute(
                    """
                    UPDATE offer_search_jobs
                    SET status = 'completed', completed_at = NOW(), updated_at = NOW(), error = NULL
                    WHERE id = %s
                    """,
                    (job_id,),
                )

    @staticmethod
    def fail_job(conn: psycopg.Connection, job_id: int, error: str) -> None:
        conn.execute(
            """
            UPDATE offer_search_jobs
            SET status = 'failed', completed_at = NOW(), updated_at = NOW(), error = %s
            WHERE id = %s
            """,
            (error[:1000], job_id),
        )


if __name__ == "__main__":
    Worker().run()
