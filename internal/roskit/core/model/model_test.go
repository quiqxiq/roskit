package model_test

import (
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
	"github.com/quiqxiq/roskit/internal/roskit/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- HotspotUser ---

func TestNewHotspotUserFromReply(t *testing.T) {
	u := model.NewHotspotUserFromReply(testhelpers.HotspotUserReply)
	require.NotNil(t, u)
	assert.Equal(t, "*1", u.ID)
	assert.Equal(t, "user001", u.Name)
	assert.Equal(t, "pass001", u.Password)
	assert.Equal(t, "2hours", u.Profile)
	assert.Equal(t, "hotspot1", u.Server)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", u.MACAddress)
	assert.Equal(t, "2h", u.LimitUptime)
	assert.Equal(t, "0", u.LimitBytesTotal)
	assert.Equal(t, "test user", u.Comment)
	assert.False(t, u.Disabled)
	assert.Equal(t, uint64(1048576), u.BytesIn)
	assert.Equal(t, uint64(2097152), u.BytesOut)
}

func TestHotspotUser_ToCacheData(t *testing.T) {
	u := model.NewHotspotUserFromReply(testhelpers.HotspotUserReply)
	ts := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	cache := u.ToCacheData(ts)

	assert.Equal(t, "*1", cache["id"])
	assert.Equal(t, "user001", cache["name"])
	assert.Equal(t, "pass001", cache["password"])
	assert.Equal(t, "2hours", cache["profile"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", cache["mac_address"])
	assert.Equal(t, "2h", cache["limit_uptime"])
	assert.Equal(t, "false", cache["disabled"])
	assert.Equal(t, "1048576", cache["bytes_in"])
	assert.Equal(t, "2097152", cache["bytes_out"])
	assert.Equal(t, "2024-03-15T10:00:00Z", cache["timestamp"])
}

func TestHotspotUser_RoundTrip_CacheData(t *testing.T) {
	original := model.NewHotspotUserFromReply(testhelpers.HotspotUserReply)
	cache := original.ToCacheData(time.Now())
	restored := model.ParseHotspotUserFromCache(cache)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.Password, restored.Password)
	assert.Equal(t, original.Profile, restored.Profile)
	assert.Equal(t, original.MACAddress, restored.MACAddress)
	assert.Equal(t, original.LimitUptime, restored.LimitUptime)
	assert.Equal(t, original.Disabled, restored.Disabled)
	assert.Equal(t, original.BytesIn, restored.BytesIn)
	assert.Equal(t, original.BytesOut, restored.BytesOut)
}

func TestNewHotspotUserFromReply_DisabledTrue(t *testing.T) {
	data := map[string]string{".id": "*2", "name": "u2", "disabled": "true"}
	u := model.NewHotspotUserFromReply(data)
	assert.True(t, u.Disabled)
}

func TestNewHotspotUserFromReply_DisabledYes(t *testing.T) {
	data := map[string]string{".id": "*3", "name": "u3", "disabled": "yes"}
	u := model.NewHotspotUserFromReply(data)
	assert.True(t, u.Disabled)
}

// --- HotspotActive ---

func TestNewHotspotActiveFromReply(t *testing.T) {
	a := model.NewHotspotActiveFromReply(testhelpers.HotspotActiveReply)
	require.NotNil(t, a)
	assert.Equal(t, "*A1", a.ID)
	assert.Equal(t, "user001", a.User)
	assert.Equal(t, "192.168.88.100", a.Address)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", a.MACAddress)
	assert.Equal(t, "hotspot1", a.Server)
	assert.Equal(t, "30m", a.Uptime)
	assert.Equal(t, "1h30m", a.SessionTimeLeft)
	assert.Equal(t, "cookie", a.LoginBy)
}

func TestHotspotActive_ToCacheData(t *testing.T) {
	a := model.NewHotspotActiveFromReply(testhelpers.HotspotActiveReply)
	cache := a.ToCacheData(time.Now())
	assert.Equal(t, "*A1", cache["id"])
	assert.Equal(t, "user001", cache["user"])
	assert.Equal(t, "192.168.88.100", cache["address"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", cache["mac_address"])
	assert.Equal(t, "1h30m", cache["session_time_left"])
}

// --- HotspotProfile ---

func TestNewHotspotProfileFromReply(t *testing.T) {
	p := model.NewHotspotProfileFromReply(testhelpers.HotspotProfileReply)
	require.NotNil(t, p)
	assert.Equal(t, "*P1", p.ID)
	assert.Equal(t, "2hours", p.Name)
	assert.Equal(t, "hs-pool", p.AddressPool)
	assert.Equal(t, "1M/1M", p.RateLimit)
	assert.Equal(t, "script", p.OnLogin)
}

func TestHotspotProfile_ToCacheData(t *testing.T) {
	p := model.NewHotspotProfileFromReply(testhelpers.HotspotProfileReply)
	cache := p.ToCacheData(time.Now())
	assert.Equal(t, "*P1", cache["id"])
	assert.Equal(t, "2hours", cache["name"])
	assert.Equal(t, "hs-pool", cache["address_pool"])
	assert.Equal(t, "1M/1M", cache["rate_limit"])
}

// --- HotspotWalledGarden ---

func TestNewHotspotWalledGardenFromReply(t *testing.T) {
	w := model.NewHotspotWalledGardenFromReply(testhelpers.HotspotWalledGardenReply)
	require.NotNil(t, w)
	assert.Equal(t, "*W1", w.ID)
	assert.Equal(t, "8.8.8.8", w.DstAddress)
	assert.Equal(t, "80", w.DstPort)
	assert.Equal(t, "tcp", w.Protocol)
	assert.Equal(t, "allow", w.Action)
	assert.False(t, w.Disabled)
}

func TestHotspotWalledGarden_ToCacheData(t *testing.T) {
	w := model.NewHotspotWalledGardenFromReply(testhelpers.HotspotWalledGardenReply)
	cache := w.ToCacheData(time.Now())
	assert.Equal(t, "*W1", cache["id"])
	assert.Equal(t, "8.8.8.8", cache["dst_address"])
	assert.Equal(t, "false", cache["disabled"])
}

// --- SystemResource ---

func TestNewSystemResourceFromReply(t *testing.T) {
	r := model.NewSystemResourceFromReply(testhelpers.SystemResourceReply)
	require.NotNil(t, r)
	assert.Equal(t, "1d2h3m", r.Uptime)
	assert.Equal(t, "7.14.3 (stable)", r.Version)
	assert.Equal(t, "arm", r.ArchitectureName)
	assert.Equal(t, "RB750Gr3", r.BoardName)
	assert.Equal(t, 12, r.CPULoad)
	assert.Equal(t, uint64(104857600), r.FreeMemory)
	assert.Equal(t, uint64(268435456), r.TotalMemory)
}

func TestSystemResource_ToCacheData(t *testing.T) {
	r := model.NewSystemResourceFromReply(testhelpers.SystemResourceReply)
	cache := r.ToCacheData(time.Now())
	assert.Equal(t, "1d2h3m", cache["uptime"])
	assert.Equal(t, "7.14.3 (stable)", cache["version"])
	assert.Equal(t, "arm", cache["architecture_name"])
	assert.Equal(t, "12", cache["cpu_load"])
	assert.Equal(t, "104857600", cache["free_memory"])
}

// --- UserCountMinusOne ---

func TestUserCountMinusOne(t *testing.T) {
	assert.Equal(t, 0, model.UserCountMinusOne(0))
	assert.Equal(t, 0, model.UserCountMinusOne(1))
	assert.Equal(t, 1, model.UserCountMinusOne(2))
	assert.Equal(t, 9, model.UserCountMinusOne(10))
}

// --- ToTags / ToFields ---

func TestHotspotUser_ToTags(t *testing.T) {
	u := model.NewHotspotUserFromReply(testhelpers.HotspotUserReply)
	tags := u.ToTags()
	assert.Equal(t, "user001", tags["name"])
	assert.Equal(t, "2hours", tags["profile"])
	assert.Equal(t, "hotspot1", tags["server"])
}

func TestHotspotUser_ToFields(t *testing.T) {
	u := model.NewHotspotUserFromReply(testhelpers.HotspotUserReply)
	fields := u.ToFields()
	assert.Equal(t, false, fields["disabled"])
	assert.Equal(t, uint64(1048576), fields["bytes_in"])
	assert.Equal(t, uint64(2097152), fields["bytes_out"])
}

// --- HotspotServer ---

func TestHotspotServer_ToCacheData(t *testing.T) {
	s := &model.HotspotServer{
		ID: "*S1", Name: "hotspot1", Interface: "ether1", Disabled: false,
	}
	cache := s.ToCacheData(time.Now())
	assert.Equal(t, "*S1", cache["id"])
	assert.Equal(t, "hotspot1", cache["name"])
	assert.Equal(t, "ether1", cache["interface"])
	assert.Equal(t, "false", cache["disabled"])
}

// --- HotspotHost ---

func TestHotspotHost_ToCacheData(t *testing.T) {
	h := &model.HotspotHost{
		ID: "*H1", MACAddress: "AA:BB:CC:DD:EE:FF", Address: "192.168.88.10",
		Server: "hotspot1", ToAddress: "10.0.0.1", Authorized: true,
	}
	cache := h.ToCacheData(time.Now())
	assert.Equal(t, "*H1", cache["id"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", cache["mac"])
	assert.Equal(t, "192.168.88.10", cache["address"])
	assert.Equal(t, "hotspot1", cache["server"])
	assert.Equal(t, "10.0.0.1", cache["to_address"])
	assert.Equal(t, "true", cache["authorized"])
}

// --- HotspotCookie ---

func TestHotspotCookie_ToCacheData(t *testing.T) {
	c := &model.HotspotCookie{
		ID: "*C1", User: "user001", MAC: "AA:BB:CC:DD:EE:FF", Address: "192.168.88.100",
	}
	cache := c.ToCacheData(time.Now())
	assert.Equal(t, "*C1", cache["id"])
	assert.Equal(t, "user001", cache["user"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", cache["mac"])
	assert.Equal(t, "192.168.88.100", cache["address"])
}

// --- IPBinding ---

func TestIPBinding_ToCacheData(t *testing.T) {
	b := &model.IPBinding{
		ID: "*B1", MAC: "AA:BB:CC:DD:EE:FF", Address: "192.168.88.50",
		Type: "bypassed", Disabled: false, Comment: "allowed device",
	}
	cache := b.ToCacheData(time.Now())
	assert.Equal(t, "*B1", cache["id"])
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", cache["mac"])
	assert.Equal(t, "192.168.88.50", cache["address"])
	assert.Equal(t, "bypassed", cache["type"])
	assert.Equal(t, "false", cache["disabled"])
	assert.Equal(t, "allowed device", cache["comment"])
}
