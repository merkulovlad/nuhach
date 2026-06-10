from urllib.parse import urlsplit
from urllib.robotparser import RobotFileParser

import httpx


class RobotsDenied(RuntimeError):
    pass


async def ensure_allowed(url: str, user_agent: str) -> None:
    """Fetch and enforce robots.txt. Network or parse failures deny access."""
    parts = urlsplit(url)
    robots_url = f"{parts.scheme}://{parts.netloc}/robots.txt"
    try:
        async with httpx.AsyncClient(timeout=10, follow_redirects=True) as client:
            response = await client.get(robots_url, headers={"User-Agent": user_agent})
            response.raise_for_status()
        parser = RobotFileParser()
        parser.set_url(robots_url)
        parser.parse(response.text.splitlines())
    except Exception as exc:
        raise RobotsDenied(f"cannot verify robots.txt for {parts.netloc}: {exc}") from exc

    if not parser.can_fetch(user_agent, url):
        raise RobotsDenied(f"robots.txt disallows {url}")
