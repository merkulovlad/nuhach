package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// SearchResponse is the JSON response for search.
type SearchResponse struct {
	Items     interface{} `json:"items"`
	RequestID string      `json:"request_id"`
	Total     int64       `json:"total"`
}

// Search handles GET /api/search.
func (h *Handler) Search(c *fiber.Ctx) error {
	query := c.Query("q", "")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "query parameter 'q' is required",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	// Optional: user ID for impression logging
	var tgID *int64

	if tgIDStr := c.Query("tg_id"); tgIDStr != "" {
		if id, err := strconv.ParseInt(tgIDStr, 10, 64); err == nil {
			tgID = &id
		}
	}

	h.logger.Info("Search request",
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	result, err := h.searchUC.Search(c.Context(), query, limit, offset, tgID)
	if err != nil {
		h.logger.Error("Search failed", zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "search failed",
		})
	}

	return c.JSON(SearchResponse{
		Items:     result.Items,
		RequestID: result.RequestID,
		Total:     result.Total,
	})
}

// GetPerfume handles GET /api/perfumes/:id.
func (h *Handler) GetPerfume(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid perfume id",
		})
	}

	perfume, err := h.searchUC.GetPerfumeByID(c.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get perfume", zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get perfume",
		})
	}

	if perfume == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "perfume not found",
		})
	}

	return c.JSON(perfume)
}

// GetSimilarPerfumes handles GET /api/perfumes/:id/similar.
func (h *Handler) GetSimilarPerfumes(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid perfume id",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// Optional: user ID for filtering
	var tgID *int64

	if tgIDStr := c.Query("tg_id"); tgIDStr != "" {
		if uid, err := strconv.ParseInt(tgIDStr, 10, 64); err == nil {
			tgID = &uid
		}
	}

	result, err := h.searchUC.GetSimilarPerfumes(c.Context(), id, limit, tgID)
	if err != nil {
		h.logger.Error("Failed to get similar perfumes", zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get similar perfumes",
		})
	}

	return c.JSON(SearchResponse{
		Items:     result.Items,
		RequestID: result.RequestID,
		Total:     result.Total,
	})
}

// VectorSearchRequest is the request body for vector search.
type VectorSearchRequest struct {
	Query     string    `json:"query"`
	Embedding []float32 `json:"embedding"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
	TgID      *int64    `json:"tg_id,omitempty"`
}

// VectorSearch handles POST /api/search/vector.
// Performs semantic search using pre-computed query embedding.
func (h *Handler) VectorSearch(c *fiber.Ctx) error {
	var req VectorSearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.Embedding) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "embedding is required",
		})
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	h.logger.Info("Vector search request",
		zap.String("query", req.Query),
		zap.Int("embedding_dim", len(req.Embedding)),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	result, err := h.searchUC.VectorSearch(c.Context(), req.Query, req.Embedding, limit, offset, req.TgID)
	if err != nil {
		h.logger.Error("Vector search failed", zap.Error(err))

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "vector search failed",
		})
	}

	return c.JSON(SearchResponse{
		Items:     result.Items,
		RequestID: result.RequestID,
		Total:     result.Total,
	})
}
