package model

import (
	"fmt"
	"strconv"
	"time"
)

func pBool(s string) bool {
	return s == "true" || s == "yes"
}

func pUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

func pInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

type HotspotUser struct {
	ID              string
	Name            string
	Password        string
	Profile         string
	Server          string
	MACAddress      string
	LimitUptime     string
	LimitBytesTotal string
	Comment         string
	Disabled        bool
	Uptime          string
	BytesIn         uint64
	BytesOut        uint64
	Timestamp       time.Time
}

func (u *HotspotUser) ToTags() map[string]string {
	return map[string]string{
		"name":    u.Name,
		"profile": u.Profile,
		"server":  u.Server,
	}
}

func (u *HotspotUser) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"disabled":          u.Disabled,
		"mac_address":       u.MACAddress,
		"limit_uptime":      u.LimitUptime,
		"limit_bytes_total": u.LimitBytesTotal,
		"comment":           u.Comment,
		"uptime":            u.Uptime,
		"bytes_in":          u.BytesIn,
		"bytes_out":         u.BytesOut,
	}
}

func (u *HotspotUser) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":                u.ID,
		"name":              u.Name,
		"password":          u.Password,
		"profile":           u.Profile,
		"server":            u.Server,
		"mac_address":       u.MACAddress,
		"limit_uptime":      u.LimitUptime,
		"limit_bytes_total": u.LimitBytesTotal,
		"comment":           u.Comment,
		"disabled":          fmt.Sprintf("%t", u.Disabled),
		"uptime":            u.Uptime,
		"bytes_in":          fmt.Sprintf("%d", u.BytesIn),
		"bytes_out":         fmt.Sprintf("%d", u.BytesOut),
		"timestamp":         ts.Format(time.RFC3339),
	}
}

type HotspotActive struct {
	ID               string
	User             string
	Address          string
	MACAddress       string
	Server           string
	Uptime           string
	SessionTimeLeft  string
	KeepaliveTimeout string
	LoginBy          string
	Timestamp        time.Time
}

func (a *HotspotActive) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":                a.ID,
		"user":              a.User,
		"address":           a.Address,
		"mac_address":       a.MACAddress,
		"server":            a.Server,
		"uptime":            a.Uptime,
		"session_time_left": a.SessionTimeLeft,
		"keepalive_timeout": a.KeepaliveTimeout,
		"login_by":          a.LoginBy,
		"timestamp":         ts.Format(time.RFC3339),
	}
}

type HotspotProfile struct {
	ID                string
	Name              string
	AddressPool       string
	RateLimit         string
	SharedUsers       string
	StatusAutoRefresh string
	OnLogin           string
	OnLogout          string
	ParentQueue       string
	TransparentProxy  bool
	OpenStatusPage    bool
	Timestamp         time.Time
}

func (p *HotspotProfile) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":                 p.ID,
		"name":               p.Name,
		"address_pool":       p.AddressPool,
		"rate_limit":         p.RateLimit,
		"shared_users":       p.SharedUsers,
		"status_autorefresh": p.StatusAutoRefresh,
		"on_login":           p.OnLogin,
		"on_logout":          p.OnLogout,
		"parent_queue":       p.ParentQueue,
		"timestamp":          ts.Format(time.RFC3339),
	}
}

type HotspotServer struct {
	ID        string
	Name      string
	Interface string
	Disabled  bool
	Timestamp time.Time
}

func (s *HotspotServer) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":        s.ID,
		"name":      s.Name,
		"interface": s.Interface,
		"disabled":  fmt.Sprintf("%t", s.Disabled),
		"timestamp": ts.Format(time.RFC3339),
	}
}

type HotspotHost struct {
	ID         string
	MACAddress string
	Address    string
	Server     string
	ToAddress  string
	Authorized bool
	Timestamp  time.Time
}

func (h *HotspotHost) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":         h.ID,
		"mac":        h.MACAddress,
		"address":    h.Address,
		"server":     h.Server,
		"to_address": h.ToAddress,
		"authorized": fmt.Sprintf("%t", h.Authorized),
		"timestamp":  ts.Format(time.RFC3339),
	}
}

type HotspotCookie struct {
	ID        string
	User      string
	MAC       string
	Address   string
	Timestamp time.Time
}

func (c *HotspotCookie) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":        c.ID,
		"user":      c.User,
		"mac":       c.MAC,
		"address":   c.Address,
		"timestamp": ts.Format(time.RFC3339),
	}
}

type IPBinding struct {
	ID        string
	MAC       string
	Address   string
	Type      string
	Disabled  bool
	Comment   string
	Timestamp time.Time
}

func (b *IPBinding) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":        b.ID,
		"mac":       b.MAC,
		"address":   b.Address,
		"type":      b.Type,
		"disabled":  fmt.Sprintf("%t", b.Disabled),
		"comment":   b.Comment,
		"timestamp": ts.Format(time.RFC3339),
	}
}

func ParseHotspotUserFromCache(data map[string]string) *HotspotUser {
	return &HotspotUser{
		ID:              data["id"],
		Name:            data["name"],
		Password:        data["password"],
		Profile:         data["profile"],
		Server:          data["server"],
		MACAddress:      data["mac_address"],
		LimitUptime:     data["limit_uptime"],
		LimitBytesTotal: data["limit_bytes_total"],
		Comment:         data["comment"],
		Disabled:        pBool(data["disabled"]),
		Uptime:          data["uptime"],
		BytesIn:         pUint64(data["bytes_in"]),
		BytesOut:        pUint64(data["bytes_out"]),
	}
}

func NewHotspotUserFromReply(data map[string]string) *HotspotUser {
	now := time.Now()
	return &HotspotUser{
		ID:              data[".id"],
		Name:            data["name"],
		Password:        data["password"],
		Profile:         data["profile"],
		Server:          data["server"],
		MACAddress:      data["mac-address"],
		LimitUptime:     data["limit-uptime"],
		LimitBytesTotal: data["limit-bytes-total"],
		Comment:         data["comment"],
		Disabled:        pBool(data["disabled"]),
		Uptime:          data["uptime"],
		BytesIn:         pUint64(data["bytes-in"]),
		BytesOut:        pUint64(data["bytes-out"]),
		Timestamp:       now,
	}
}

func NewHotspotActiveFromReply(data map[string]string) *HotspotActive {
	now := time.Now()
	return &HotspotActive{
		ID:               data[".id"],
		User:             data["user"],
		Address:          data["address"],
		MACAddress:       data["mac-address"],
		Server:           data["server"],
		Uptime:           data["uptime"],
		SessionTimeLeft:  data["session-time-left"],
		KeepaliveTimeout: data["keepalive-timeout"],
		LoginBy:          data["login-by"],
		Timestamp:        now,
	}
}

func NewHotspotProfileFromReply(data map[string]string) *HotspotProfile {
	now := time.Now()
	return &HotspotProfile{
		ID:                data[".id"],
		Name:              data["name"],
		AddressPool:       data["address-pool"],
		RateLimit:         data["rate-limit"],
		SharedUsers:       data["shared-users"],
		StatusAutoRefresh: data["status-autorefresh"],
		OnLogin:           data["on-login"],
		OnLogout:          data["on-logout"],
		ParentQueue:       data["parent-queue"],
		Timestamp:         now,
	}
}

func UserCountMinusOne(count int) int {
	if count > 1 {
		return count - 1
	}
	return 0
}

type HotspotWalledGarden struct {
	ID         string
	DstAddress string
	DstPort    string
	SrcAddress string
	Protocol   string
	Action     string
	Comment    string
	Disabled   bool
	Server     string
	Timestamp  time.Time
}

func NewHotspotWalledGardenFromReply(data map[string]string) *HotspotWalledGarden {
	now := time.Now()
	return &HotspotWalledGarden{
		ID:         data[".id"],
		DstAddress: data["dst-address"],
		DstPort:    data["dst-port"],
		SrcAddress: data["src-address"],
		Protocol:   data["protocol"],
		Action:     data["action"],
		Comment:    data["comment"],
		Disabled:   pBool(data["disabled"]),
		Server:     data["server"],
		Timestamp:  now,
	}
}

func (w *HotspotWalledGarden) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":          w.ID,
		"dst_address": w.DstAddress,
		"dst_port":    w.DstPort,
		"src_address": w.SrcAddress,
		"protocol":    w.Protocol,
		"action":      w.Action,
		"comment":     w.Comment,
		"disabled":    fmt.Sprintf("%t", w.Disabled),
		"server":      w.Server,
		"timestamp":   ts.Format(time.RFC3339),
	}
}
