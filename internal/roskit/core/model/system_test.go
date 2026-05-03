package model_test

import (
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemSchedulerFromReply(t *testing.T) {
	data := map[string]string{
		".id":         "*S1",
		"name":        "daily-backup",
		"start-date":  "jan/01/2024",
		"start-time":  "03:00:00",
		"interval":    "24h",
		"next-run":    "jan/02/2024 03:00:00",
		"on-event":    "/system backup save name=auto",
		"disabled":    "false",
		"comment":     "auto backup",
		"owner":       "admin",
		"policy":      "read,write,policy,test",
	}
	s := model.NewSystemSchedulerFromReply(data)
	require.NotNil(t, s)
	assert.Equal(t, "*S1", s.ID)
	assert.Equal(t, "daily-backup", s.Name)
	assert.Equal(t, "jan/01/2024", s.StartDate)
	assert.Equal(t, "03:00:00", s.StartTime)
	assert.Equal(t, "24h", s.Interval)
	assert.Equal(t, "jan/02/2024 03:00:00", s.NextRun)
	assert.Equal(t, "/system backup save name=auto", s.OnEvent)
	assert.False(t, s.Disabled)
	assert.Equal(t, "auto backup", s.Comment)
	assert.Equal(t, "admin", s.Owner)
	assert.Equal(t, "read,write,policy,test", s.Policy)
}

func TestSystemScheduler_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id": "*S1", "name": "daily-backup", "start-date": "jan/01/2024",
		"start-time": "03:00:00", "interval": "24h", "next-run": "jan/02/2024 03:00:00",
		"on-event": "/system backup save", "disabled": "false", "comment": "backup",
		"owner": "admin", "policy": "read,write",
	}
	s := model.NewSystemSchedulerFromReply(data)
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := s.ToCacheData(ts)

	assert.Equal(t, "*S1", cache["id"])
	assert.Equal(t, "daily-backup", cache["name"])
	assert.Equal(t, "jan/01/2024", cache["start_date"])
	assert.Equal(t, "03:00:00", cache["start_time"])
	assert.Equal(t, "24h", cache["interval"])
	assert.Equal(t, "jan/02/2024 03:00:00", cache["next_run"])
	assert.Equal(t, "/system backup save", cache["on_event"])
	assert.Equal(t, "false", cache["disabled"])
	assert.Equal(t, "backup", cache["comment"])
	assert.Equal(t, "admin", cache["owner"])
	assert.Equal(t, "read,write", cache["policy"])
	assert.Equal(t, "2024-06-15T12:00:00Z", cache["timestamp"])
}

func TestNewSystemSchedulerFromReply_DisabledTrue(t *testing.T) {
	data := map[string]string{".id": "*S2", "name": "off", "disabled": "true"}
	s := model.NewSystemSchedulerFromReply(data)
	assert.True(t, s.Disabled)
}

func TestNewSystemScriptFromReply(t *testing.T) {
	data := map[string]string{
		".id":          "*SC1",
		"name":         "on-login",
		"source":       ":put (\",1,2h,1000,,,\")",
		"owner":        "admin",
		"comment":      "mikhmon",
		"policy":       "read,write,policy,test",
		"last-started": "jan/15/2024 10:30:00",
		"run-count":    "42",
	}
	s := model.NewSystemScriptFromReply(data)
	require.NotNil(t, s)
	assert.Equal(t, "*SC1", s.ID)
	assert.Equal(t, "on-login", s.Name)
	assert.Equal(t, ":put (\",1,2h,1000,,,\")", s.Source)
	assert.Equal(t, "admin", s.Owner)
	assert.Equal(t, "mikhmon", s.Comment)
	assert.Equal(t, "read,write,policy,test", s.Policy)
	assert.Equal(t, "jan/15/2024 10:30:00", s.LastStarted)
	assert.Equal(t, int64(42), s.RunCount)
}

func TestSystemScript_ToCacheData(t *testing.T) {
	data := map[string]string{
		".id": "*SC1", "name": "on-login", "source": ":put hello",
		"owner": "admin", "comment": "test", "policy": "read",
		"last-started": "jan/15/2024 10:30:00", "run-count": "5",
	}
	s := model.NewSystemScriptFromReply(data)
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := s.ToCacheData(ts)

	assert.Equal(t, "*SC1", cache["id"])
	assert.Equal(t, "on-login", cache["name"])
	assert.Equal(t, ":put hello", cache["source"])
	assert.Equal(t, "admin", cache["owner"])
	assert.Equal(t, "test", cache["comment"])
	assert.Equal(t, "read", cache["policy"])
	assert.Equal(t, "jan/15/2024 10:30:00", cache["last_started"])
	assert.Equal(t, "5", cache["run_count"])
	assert.Equal(t, "2024-06-15T12:00:00Z", cache["timestamp"])
}

func TestNewSystemIdentityFromReply(t *testing.T) {
	data := map[string]string{
		"name": "MikroTik-GW01",
	}
	i := model.NewSystemIdentityFromReply(data)
	require.NotNil(t, i)
	assert.Equal(t, "MikroTik-GW01", i.Name)
}

func TestSystemIdentity_ToCacheData(t *testing.T) {
	data := map[string]string{"name": "MikroTik-GW01"}
	i := model.NewSystemIdentityFromReply(data)
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := i.ToCacheData(ts)

	assert.Equal(t, "MikroTik-GW01", cache["name"])
	assert.Equal(t, "2024-06-15T12:00:00Z", cache["timestamp"])
}

func TestNewSystemClockFromReply(t *testing.T) {
	data := map[string]string{
		"time":            "10:30:45",
		"date":            "jan/15/2024",
		"time-zone-name":  "Asia/Jakarta",
		"dst-active":      "false",
	}
	c := model.NewSystemClockFromReply(data)
	require.NotNil(t, c)
	assert.Equal(t, "10:30:45", c.Time)
	assert.Equal(t, "jan/15/2024", c.Date)
	assert.Equal(t, "Asia/Jakarta", c.TimeZoneName)
	assert.False(t, c.DSTActive)
}

func TestSystemClock_ToCacheData(t *testing.T) {
	data := map[string]string{
		"time": "10:30:45", "date": "jan/15/2024",
		"time-zone-name": "Asia/Jakarta", "dst-active": "false",
	}
	c := model.NewSystemClockFromReply(data)
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := c.ToCacheData(ts)

	assert.Equal(t, "10:30:45", cache["time"])
	assert.Equal(t, "jan/15/2024", cache["date"])
	assert.Equal(t, "Asia/Jakarta", cache["time_zone_name"])
	assert.Equal(t, "false", cache["dst_active"])
	assert.Equal(t, "2024-06-15T12:00:00Z", cache["timestamp"])
}

func TestNewSystemClockFromReply_DSTActive(t *testing.T) {
	data := map[string]string{
		"time": "10:30:45", "date": "jun/15/2024",
		"time-zone-name": "Europe/London", "dst-active": "true",
	}
	c := model.NewSystemClockFromReply(data)
	assert.True(t, c.DSTActive)
}

func TestNewSystemHealthFromReply(t *testing.T) {
	data := map[string]string{
		"voltage":            "24.5",
		"temperature":        "42",
		"cpu-temperature":    "58",
		"power-consumption":  "10.3",
		"board-temperature":  "38",
	}
	h := model.NewSystemHealthFromReply(data)
	require.NotNil(t, h)
	assert.Equal(t, "24.5", h.Voltage)
	assert.Equal(t, "42", h.Temperature)
	assert.Equal(t, "58", h.CPUTemperature)
	assert.Equal(t, "10.3", h.PowerConsumption)
	assert.Equal(t, "38", h.BoardTemperature)
}

func TestSystemHealth_ToCacheData(t *testing.T) {
	data := map[string]string{
		"voltage": "24.5", "temperature": "42",
		"cpu-temperature": "58", "power-consumption": "10.3",
		"board-temperature": "38",
	}
	h := model.NewSystemHealthFromReply(data)
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := h.ToCacheData(ts)

	assert.Equal(t, "24.5", cache["voltage"])
	assert.Equal(t, "42", cache["temperature"])
	assert.Equal(t, "58", cache["cpu_temperature"])
	assert.Equal(t, "10.3", cache["power_consumption"])
	assert.Equal(t, "38", cache["board_temperature"])
	assert.Equal(t, "2024-06-15T12:00:00Z", cache["timestamp"])
}

func TestNewSystemRouterboardFromReply(t *testing.T) {
	data := map[string]string{
		"model":            "RB750Gr3",
		"serial-number":    "ABC12345678",
		"firmware-type":    "ar7161",
		"firmware":         "3.41",
		"revision":         "r2",
		"upgrade-firmware": "3.42",
		"routerboard":      "true",
	}
	r := model.NewSystemRouterboardFromReply(data)
	require.NotNil(t, r)
	assert.Equal(t, "RB750Gr3", r.Model)
	assert.Equal(t, "ABC12345678", r.SerialNumber)
	assert.Equal(t, "ar7161", r.FirmwareType)
	assert.Equal(t, "3.41", r.Firmware)
	assert.Equal(t, "r2", r.Revision)
	assert.Equal(t, "3.42", r.UpgradeFirmware)
	assert.True(t, r.Routerboard)
}

func TestSystemRouterboard_ToCacheData(t *testing.T) {
	data := map[string]string{
		"model": "RB750Gr3", "serial-number": "ABC12345678",
		"firmware-type": "ar7161", "firmware": "3.41",
		"revision": "r2", "upgrade-firmware": "3.42",
		"routerboard": "true",
	}
	r := model.NewSystemRouterboardFromReply(data)
	ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	cache := r.ToCacheData(ts)

	assert.Equal(t, "RB750Gr3", cache["model"])
	assert.Equal(t, "ABC12345678", cache["serial_number"])
	assert.Equal(t, "ar7161", cache["firmware_type"])
	assert.Equal(t, "3.41", cache["firmware"])
	assert.Equal(t, "r2", cache["revision"])
	assert.Equal(t, "3.42", cache["upgrade_firmware"])
	assert.Equal(t, "true", cache["routerboard"])
	assert.Equal(t, "2024-06-15T12:00:00Z", cache["timestamp"])
}

func TestNewSystemRouterboardFromReply_NotRouterboard(t *testing.T) {
	data := map[string]string{
		"model": "CHR", "routerboard": "false",
	}
	r := model.NewSystemRouterboardFromReply(data)
	assert.False(t, r.Routerboard)
}
