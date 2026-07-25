package middlewares

import (
	"testing"

	"firebase.google.com/go/auth"
	"github.com/stretchr/testify/assert"
)

func TestFirebaseAuthContextRejectsIncompleteIdentity(t *testing.T) {
	user, ok := firebaseAuthContext(&auth.Token{UID: "user-1", Claims: map[string]interface{}{}})
	assert.False(t, ok)
	assert.True(t, user.IsNoop())

	user, ok = firebaseAuthContext(&auth.Token{
		UID:    "user-1",
		Claims: map[string]interface{}{"email": "user@example.com"},
	})
	assert.True(t, ok)
	assert.Equal(t, "user-1", user.ID.String())
	assert.Equal(t, "user@example.com", user.Email)
}
