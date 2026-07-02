package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDueDate_DateOnlyFormatUses2359(t *testing.T) {
	due := parseDueDate("2025-03-15")
	require.NotNil(t, due)
	assert.Equal(t, 23, due.Hour())
	assert.Equal(t, 59, due.Minute())
	assert.Equal(t, time.March, due.Month())
	assert.Equal(t, 2025, due.Year())
}

func TestParseDueDate_DateOnlyShortFormatUses2359(t *testing.T) {
	due := parseDueDate("03-15")
	require.NotNil(t, due)
	assert.Equal(t, 23, due.Hour())
	assert.Equal(t, 59, due.Minute())
	assert.Equal(t, time.March, due.Month())
	assert.Equal(t, time.Now().Year(), due.Year())
}

func TestParseDueDate_DateTimeFormatPreservesTime(t *testing.T) {
	due := parseDueDate("2025-03-15 14:30")
	require.NotNil(t, due)
	assert.Equal(t, 14, due.Hour())
	assert.Equal(t, 30, due.Minute())
	assert.Equal(t, time.March, due.Month())
	assert.Equal(t, 2025, due.Year())
}

func TestParseDueDate_SlashDateFormatUses2359(t *testing.T) {
	due := parseDueDate("2025/03/15")
	require.NotNil(t, due)
	assert.Equal(t, 23, due.Hour())
	assert.Equal(t, 59, due.Minute())
}

func TestParseDueDate_InvalidStringReturnsNil(t *testing.T) {
	due := parseDueDate("not-a-date")
	assert.Nil(t, due)
}

func TestParseDueDate_EmptyStringReturnsNil(t *testing.T) {
	due := parseDueDate("")
	assert.Nil(t, due)
}
