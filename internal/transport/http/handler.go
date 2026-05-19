// Package http contains Fiber HTTP handlers and middleware.
package http

import (
	"strconv"
	"time"

	"github.com/merkulovlad/nuhach/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Handler holds all HTTP handlers.
type Handler struct {
	searchUC *usecase.SearchUseCase
	recsUC   *usecase.RecommendationUseCase
	eventUC  *usecase.EventUseCase
	logger   *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(
	searchUC *usecase.SearchUseCase,
	recsUC *usecase.RecommendationUseCase,
	eventUC *usecase.EventUseCase,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		searchUC: searchUC,
		recsUC:   recsUC,
		eventUC:  eventUC,
		logger:   logger,
	}
}

// RegisterRoutes registers all HTTP routes.
func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Health check
	api.Get("/health", h.HealthCheck)

	// Search
	api.Get("/search", h.Search)
	api.Post("/search/vector", h.VectorSearch)

	// Perfumes
	api.Get("/perfumes/:id", h.GetPerfume)
	api.Get("/perfumes/:id/similar", h.GetSimilarPerfumes)

	// User events
	api.Post("/users/:tg_id/events", h.CreateEvent)

	// Recommendations
	api.Get("/users/:tg_id/recommendations", h.GetRecommendations)

	// User saves
	api.Get("/users/:tg_id/saves", h.GetUserSaves)
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// parseIntParam parses an integer query parameter.
func parseIntParam(c *fiber.Ctx, name string, defaultValue int) int {
	val := c.Query(name)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// parseTgID parses the tg_id path parameter.
func parseTgID(c *fiber.Ctx) int64 {
	tgIDStr := c.Params("tg_id")
	tgID, _ := strconv.ParseInt(tgIDStr, 10, 64)
	return tgID
}
