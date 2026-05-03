package service_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSalesRecordFromScript_Valid(t *testing.T) {
	name := "jan/01/2024-|-10:30:00-|-user001-|-5000-|-192.168.1.100-|-AA:BB:CC:DD:EE:FF-|-30d-|-2hours-|-"
	rec := service.ParseSalesRecordFromScript(name, "jan2024", "jan/01/2024", "mikhmon")
	require.NotNil(t, rec)
	assert.Equal(t, "jan/01/2024", rec.Date)
	assert.Equal(t, "10:30:00", rec.Time)
	assert.Equal(t, "user001", rec.Username)
	assert.Equal(t, "5000", rec.Price)
	assert.Equal(t, "192.168.1.100", rec.IPAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", rec.MACAddress)
	assert.Equal(t, "30d", rec.Validity)
	assert.Equal(t, "2hours", rec.Profile)
	assert.Equal(t, "jan2024", rec.Owner)
	assert.Equal(t, "jan/01/2024", rec.Source)
}

func TestParseSalesRecordFromScript_WithComment(t *testing.T) {
	name := "jan/01/2024-|-10:30:00-|-user001-|-5000-|-192.168.1.100-|-AA:BB:CC:DD:EE:FF-|-30d-|-2hours-|-special note"
	rec := service.ParseSalesRecordFromScript(name, "jan2024", "jan/01/2024", "mikhmon")
	require.NotNil(t, rec)
	assert.Equal(t, "special note", rec.Comment)
}

func TestParseSalesRecordFromScript_TooFewParts(t *testing.T) {
	name := "jan/01/2024-|-10:30:00-|-user001"
	rec := service.ParseSalesRecordFromScript(name, "jan2024", "jan/01/2024", "mikhmon")
	assert.Nil(t, rec, "fewer than 8 parts should return nil")
}

func TestParseSalesRecordFromScript_Empty(t *testing.T) {
	rec := service.ParseSalesRecordFromScript("", "", "", "")
	assert.Nil(t, rec)
}
