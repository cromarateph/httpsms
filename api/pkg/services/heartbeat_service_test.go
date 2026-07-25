package services

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	"github.com/nyaruka/phonenumbers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type heartbeatRepositoryStub struct {
	repositories.HeartbeatRepository
	last func(context.Context, entities.UserID, string) (*entities.Heartbeat, error)
}

func (stub *heartbeatRepositoryStub) Last(ctx context.Context, userID entities.UserID, owner string) (*entities.Heartbeat, error) {
	return stub.last(ctx, userID, owner)
}

type phoneRepositoryStub struct {
	repositories.PhoneRepository
	load func(context.Context, entities.UserID, string) (*entities.Phone, error)
}

func (stub *phoneRepositoryStub) Load(ctx context.Context, userID entities.UserID, owner string) (*entities.Phone, error) {
	return stub.load(ctx, userID, owner)
}

func newHeartbeatServiceForTest(repository repositories.HeartbeatRepository) *HeartbeatService {
	logger := &noopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	return NewHeartbeatService(logger, tracer, repository, nil, nil)
}

func TestPhoneIsOnlineWithoutMonitorWhenHeartbeatIsFresh(t *testing.T) {
	service := newHeartbeatServiceForTest(&heartbeatRepositoryStub{
		last: func(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
			return &entities.Heartbeat{Timestamp: time.Now().UTC().Add(-31 * time.Minute)}, nil
		},
	})

	assert.True(t, service.PhoneIsOnline(context.Background(), "user-id", "+18005550199"))
}

func TestPhoneIsOfflineWhenHeartbeatIsStaleOrMissing(t *testing.T) {
	tests := []struct {
		name string
		last func(context.Context, entities.UserID, string) (*entities.Heartbeat, error)
	}{
		{
			name: "stale",
			last: func(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
				return &entities.Heartbeat{Timestamp: time.Now().UTC().Add(-33 * time.Minute)}, nil
			},
		},
		{
			name: "missing",
			last: func(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
				return nil, stacktrace.NewErrorWithCodef(repositories.ErrCodeNotFound, "not found")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newHeartbeatServiceForTest(&heartbeatRepositoryStub{last: test.last})
			assert.False(t, service.PhoneIsOnline(context.Background(), "user-id", "+18005550199"))
		})
	}
}

func TestSendMessageRejectsStalePhoneBeforePersistence(t *testing.T) {
	logger := &noopLogger{}
	tracer := telemetry.NewOtelLogger("test", logger)
	heartbeats := newHeartbeatServiceForTest(&heartbeatRepositoryStub{
		last: func(context.Context, entities.UserID, string) (*entities.Heartbeat, error) {
			return &entities.Heartbeat{Timestamp: time.Now().UTC().Add(-33 * time.Minute)}, nil
		},
	})
	phones := NewPhoneService(logger, tracer, &phoneRepositoryStub{
		load: func(_ context.Context, userID entities.UserID, owner string) (*entities.Phone, error) {
			return &entities.Phone{UserID: userID, PhoneNumber: owner, SIM: entities.SIM1}, nil
		},
	}, heartbeats, nil)
	service := NewMessageService(logger, tracer, nil, nil, phones, nil, "")
	owner, err := phonenumbers.Parse("+18005550199", phonenumbers.UNKNOWN_REGION)
	require.NoError(t, err)

	_, err = service.SendMessage(context.Background(), MessageSendParams{
		Owner:             owner,
		Contact:           "+18005550100",
		Content:           "hello",
		UserID:            "user-id",
		RequestReceivedAt: time.Now().UTC(),
	})

	require.ErrorIs(t, err, ErrPhoneOffline)
}
