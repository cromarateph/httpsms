package repositories

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestOutstandingMessageClaimQuery(t *testing.T) {
	db, err := gorm.Open(
		postgres.New(postgres.Config{Conn: &messageThreadTestConnPool{}}),
		&gorm.Config{DisableAutomaticPing: true, DryRun: true},
	)
	require.NoError(t, err)

	t.Run("fallback claims only due scheduled or expired messages", func(t *testing.T) {
		result := outstandingMessageClaimQuery(
			db,
			&entities.Message{},
			entities.UserID("user-id"),
			uuid.Nil,
			[]string{"+18005550199"},
		).Update("status", entities.MessageStatusSending)

		query := result.Statement.SQL.String()
		assert.Contains(t, query, "notification_scheduled_at <= CURRENT_TIMESTAMP")
		assert.Contains(t, query, "ORDER BY notification_scheduled_at ASC, created_at ASC")
		assert.GreaterOrEqual(t, strings.Count(query, "status IN"), 2)
		assert.NotContains(t, fmt.Sprint(result.Statement.Vars), string(entities.MessageStatusPending))
	})

	t.Run("exact FCM claim retains pending status support", func(t *testing.T) {
		result := outstandingMessageClaimQuery(
			db,
			&entities.Message{},
			entities.UserID("user-id"),
			uuid.New(),
			[]string{"+18005550199"},
		).Update("status", entities.MessageStatusSending)

		assert.NotContains(t, result.Statement.SQL.String(), "notification_scheduled_at")
		assert.Contains(t, fmt.Sprint(result.Statement.Vars), string(entities.MessageStatusPending))
	})
}
