package http

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merkulovlad/nuhach/internal/usecase"
	"go.uber.org/zap"
)

// GetOffers returns cached offers and the current search state.
func (h *Handler) GetOffers(c *fiber.Ctx) error {
	perfumeID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid perfume id"})
	}

	result, err := h.offerUC.Get(c.Context(), perfumeID)
	if err != nil {
		return h.handleOfferError(c, err)
	}

	return c.JSON(result)
}

// SearchOffers queues a scraper job unless fresh cached offers already exist.
func (h *Handler) SearchOffers(c *fiber.Ctx) error {
	perfumeID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid perfume id"})
	}

	force := c.QueryBool("force", false)

	result, err := h.offerUC.Search(c.Context(), perfumeID, force)
	if err != nil {
		return h.handleOfferError(c, err)
	}

	if result.Status == "searching" || result.Status == "refreshing" {
		return c.Status(fiber.StatusAccepted).JSON(result)
	}

	return c.JSON(result)
}

func (h *Handler) handleOfferError(c *fiber.Ctx, err error) error {
	if errors.Is(err, usecase.ErrPerfumeNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "perfume not found"})
	}

	h.logger.Error("Offer request failed", zap.Error(err))

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "offer request failed"})
}
