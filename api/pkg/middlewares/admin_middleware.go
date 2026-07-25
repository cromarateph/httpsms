package middlewares

import (
	"strings"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/gofiber/fiber/v3"
)

// ContextKeyFirebaseAuthUser stores the identity verified from a Firebase bearer token.
const ContextKeyFirebaseAuthUser = "auth.firebase.user"

// Admin restricts a route to Firebase-authenticated users in the email allowlist.
func Admin(allowedEmails []string) fiber.Handler {
	allowed := make(map[string]struct{}, len(allowedEmails))
	for _, email := range allowedEmails {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
			allowed[email] = struct{}{}
		}
	}

	return func(c fiber.Ctx) error {
		user, ok := c.Locals(ContextKeyFirebaseAuthUser).(entities.AuthContext)
		if !ok || user.IsNoop() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Administrator access requires a verified Firebase login.",
			})
		}

		if _, ok = allowed[strings.ToLower(user.Email)]; !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "You do not have administrator access.",
			})
		}

		return c.Next()
	}
}
