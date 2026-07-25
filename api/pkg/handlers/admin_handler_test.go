package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAdminReportPeriodUsesInclusiveDates(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	from, to, validation := adminReportPeriod("2026-07-01", "2026-07-03", now)

	assert.Empty(t, validation)
	assert.Equal(t, "2026-07-01", from.Format(time.DateOnly))
	assert.Equal(t, "2026-07-04", to.Format(time.DateOnly))
}
