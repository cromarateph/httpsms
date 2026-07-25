package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFillAdminDailyUsageIncludesEmptyDays(t *testing.T) {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 3)
	rows := []adminDailyRow{{
		Day:      from.AddDate(0, 0, 1),
		Sent:     3,
		Received: 2,
	}}

	assert.Equal(t, []AdminDailyUsage{
		{Date: "2026-07-01"},
		{Date: "2026-07-02", Sent: 3, Received: 2, Total: 5},
		{Date: "2026-07-03"},
	}, fillAdminDailyUsage(from, to, rows))
}
