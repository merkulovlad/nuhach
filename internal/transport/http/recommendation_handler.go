package http

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// RecommendationResponse is the JSON response for recommendations.
type RecommendationResponse struct {
	Items          interface{} `json:"items"`
	RequestID      string      `json:"request_id"`
	ExplorationIDs []int64     `json:"exploration_ids,omitempty"`
}

// GetRecommendations handles GET /api/users/:tg_id/recommendations.
func (h *Handler) GetRecommendations(c *fiber.Ctx) error {
	tgID, err := strconv.ParseInt(c.Params("tg_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tg_id",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	h.logger.Info("Recommendations request",
		zap.Int64("tg_id", tgID),
		zap.Int("limit", limit),
	)

	result, err := h.recsUC.GetRecommendations(c.Context(), tgID, limit)
	if err != nil {
		h.logger.Error("Failed to get recommendations", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get recommendations",
		})
	}

	return c.JSON(RecommendationResponse{
		Items:          result.Items,
		RequestID:      result.RequestID,
		ExplorationIDs: result.ExplorationIDs,
	})
}
