package http

import (
	"strconv"

	"github.com/merkulovlad/nuhach/internal/domain"
	"github.com/merkulovlad/nuhach/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// CreateEventRequest is the JSON request for creating an event.
type CreateEventRequest struct {
	PerfumeID int64  `json:"perfume_id"`
	EventType string `json:"event_type"`
	Rating    *int   `json:"rating,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// CreateEvent handles POST /api/users/:tg_id/events.
func (h *Handler) CreateEvent(c *fiber.Ctx) error {
	tgID, err := strconv.ParseInt(c.Params("tg_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tg_id",
		})
	}

	var req CreateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate event type
	eventType := domain.EventType(req.EventType)
	switch eventType {
	case domain.EventImpression, domain.EventClick, domain.EventLike,
		domain.EventDislike, domain.EventSave, domain.EventMySaves:
		// Valid event types
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event_type",
			"valid_types": []string{
				"impression", "click", "like", "dislike", "save", "my_saves",
			},
		})
	}

	// Validate rating if provided
	if req.Rating != nil && (*req.Rating < 1 || *req.Rating > 5) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "rating must be between 1 and 5",
		})
	}

	// Special handling for my_saves event
	if eventType == domain.EventMySaves {
		saves, err := h.eventUC.GetUserSaves(c.Context(), tgID)
		if err != nil {
			h.logger.Error("Failed to get user saves", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get user saves",
			})
		}
		return c.JSON(fiber.Map{
			"items": saves,
		})
	}

	h.logger.Info("Creating event",
		zap.Int64("tg_id", tgID),
		zap.Int64("perfume_id", req.PerfumeID),
		zap.String("event_type", req.EventType),
	)

	err = h.eventUC.CreateEvent(c.Context(), usecase.CreateEventRequest{
		TgID:      tgID,
		PerfumeID: req.PerfumeID,
		EventType: eventType,
		Rating:    req.Rating,
		RequestID: req.RequestID,
	})
	if err != nil {
		h.logger.Error("Failed to create event", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create event",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
	})
}

// GetUserSaves handles GET /api/users/:tg_id/saves.
func (h *Handler) GetUserSaves(c *fiber.Ctx) error {
	tgID, err := strconv.ParseInt(c.Params("tg_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid tg_id",
		})
	}

	saves, err := h.eventUC.GetUserSaves(c.Context(), tgID)
	if err != nil {
		h.logger.Error("Failed to get user saves", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get user saves",
		})
	}

	return c.JSON(fiber.Map{
		"items": saves,
	})
}
