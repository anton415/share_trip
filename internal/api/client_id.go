package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j.ru/share-trip/internal/middleware"
)

func clientIDFromClaims(c *fiber.Ctx) (uuid.UUID, error) {
	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	clientID, err := uuid.Parse(claims.Subject)
	if err != nil || clientID == uuid.Nil {
		return uuid.Nil, fiber.NewError(
			fiber.StatusUnauthorized,
			"invalid token subject",
		)
	}

	return clientID, nil
}
