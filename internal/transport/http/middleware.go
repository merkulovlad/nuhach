package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests.
func LoggingMiddleware(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		logger.Info("HTTP Request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.IP()),
		)

		return err
	}
}

// RecoveryMiddleware recovers from panics.
func RecoveryMiddleware(logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Path()),
				)

				if err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				}); err != nil {
					logger.Error("Failed to write panic response", zap.Error(err))
				}
			}
		}()

		return c.Next()
	}
}
