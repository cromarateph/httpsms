package middlewares

import (
	"net/http/httptest"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminRequiresAllowlistedFirebaseUser(t *testing.T) {
	app := fiber.New()
	app.Get("/allowed", func(c fiber.Ctx) error {
		c.Locals(ContextKeyFirebaseAuthUser, entities.AuthContext{ID: "1", Email: "ADMIN@example.com"})
		return c.Next()
	}, Admin([]string{"admin@example.com"}), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	app.Get("/denied", func(c fiber.Ctx) error {
		c.Locals(ContextKeyFirebaseAuthUser, entities.AuthContext{ID: "2", Email: "user@example.com"})
		return c.Next()
	}, Admin([]string{"admin@example.com"}), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	allowed, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/allowed", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, allowed.StatusCode)

	denied, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/denied", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, denied.StatusCode)
}
