package validators

import (
	"context"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMessageOutstandingOptionalID(t *testing.T) {
	request := requests.MessageOutstanding{}

	assert.Empty(t, MessageHandlerValidator{}.ValidateMessageOutstanding(context.Background(), request))
	assert.Equal(t, uuid.Nil, request.ToGetOutstandingParams("", entities.AuthContext{}, time.Time{}).MessageID)
	assert.NotEmpty(t, MessageHandlerValidator{}.ValidateMessageOutstanding(
		context.Background(),
		requests.MessageOutstanding{MessageID: "invalid"},
	))
}
