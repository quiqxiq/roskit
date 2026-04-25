package model

import (
	"fmt"
	"time"
)

type SystemResource struct {
	Uptime           string
	Version          string
	ArchitectureName string
	BoardName        string
	CPU              string
	CPULoad          int
	FreeMemory       uint64
	TotalMemory      uint64
	FreeHDDSpace     uint64
	TotalHDDSpace    uint64
	Timestamp        time.Time
}

func (s *SystemResource) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"uptime":            s.Uptime,
		"version":           s.Version,
		"architecture_name": s.ArchitectureName,
		"board_name":        s.BoardName,
		"cpu":               s.CPU,
		"cpu_load":          fmt.Sprintf("%d", s.CPULoad),
		"free_memory":       fmt.Sprintf("%d", s.FreeMemory),
		"total_memory":      fmt.Sprintf("%d", s.TotalMemory),
		"free_hdd_space":    fmt.Sprintf("%d", s.FreeHDDSpace),
		"total_hdd_space":   fmt.Sprintf("%d", s.TotalHDDSpace),
		"timestamp":         ts.Format(time.RFC3339),
	}
}

func NewSystemResourceFromReply(data map[string]string) *SystemResource {
	now := time.Now()
	return &SystemResource{
		Uptime:           data["uptime"],
		Version:          data["version"],
		ArchitectureName: data["architecture-name"],
		BoardName:        data["board-name"],
		CPU:              data["cpu"],
		CPULoad:          int(pInt64(data["cpu-load"])),
		FreeMemory:       pUint64(data["free-memory"]),
		TotalMemory:      pUint64(data["total-memory"]),
		FreeHDDSpace:     pUint64(data["free-hdd-space"]),
		TotalHDDSpace:    pUint64(data["total-hdd-space"]),
		Timestamp:        now,
	}
}

type SystemScheduler struct {
	ID        string
	Name      string
	StartDate string
	StartTime string
	Interval  string
	NextRun   string
	OnEvent   string
	Disabled  bool
	Comment   string
	Owner     string
	Policy    string
	Timestamp time.Time
}

func NewSystemSchedulerFromReply(data map[string]string) *SystemScheduler {
	now := time.Now()
	return &SystemScheduler{
		ID:        data[".id"],
		Name:      data["name"],
		StartDate: data["start-date"],
		StartTime: data["start-time"],
		Interval:  data["interval"],
		NextRun:   data["next-run"],
		OnEvent:   data["on-event"],
		Disabled:  pBool(data["disabled"]),
		Comment:   data["comment"],
		Owner:     data["owner"],
		Policy:    data["policy"],
		Timestamp: now,
	}
}

func (s *SystemScheduler) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":         s.ID,
		"name":       s.Name,
		"start_date": s.StartDate,
		"start_time": s.StartTime,
		"interval":   s.Interval,
		"next_run":   s.NextRun,
		"on_event":   s.OnEvent,
		"disabled":   fmt.Sprintf("%t", s.Disabled),
		"comment":    s.Comment,
		"owner":      s.Owner,
		"policy":     s.Policy,
		"timestamp":  ts.Format(time.RFC3339),
	}
}

type SystemScript struct {
	ID          string
	Name        string
	Source      string
	Owner       string
	Comment     string
	Policy      string
	LastStarted string
	RunCount    int64
	Timestamp   time.Time
}

func NewSystemScriptFromReply(data map[string]string) *SystemScript {
	now := time.Now()
	return &SystemScript{
		ID:          data[".id"],
		Name:        data["name"],
		Source:      data["source"],
		Owner:       data["owner"],
		Comment:     data["comment"],
		Policy:      data["policy"],
		LastStarted: data["last-started"],
		RunCount:    pInt64(data["run-count"]),
		Timestamp:   now,
	}
}

func (s *SystemScript) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":           s.ID,
		"name":         s.Name,
		"source":       s.Source,
		"owner":        s.Owner,
		"comment":      s.Comment,
		"policy":       s.Policy,
		"last_started": s.LastStarted,
		"run_count":    fmt.Sprintf("%d", s.RunCount),
		"timestamp":    ts.Format(time.RFC3339),
	}
}

type SystemIdentity struct {
	Name      string
	Timestamp time.Time
}

func NewSystemIdentityFromReply(data map[string]string) *SystemIdentity {
	return &SystemIdentity{
		Name:      data["name"],
		Timestamp: time.Now(),
	}
}

func (i *SystemIdentity) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"name":      i.Name,
		"timestamp": ts.Format(time.RFC3339),
	}
}

type SystemClock struct {
	Time         string
	Date         string
	TimeZoneName string
	DSTActive    bool
	Timestamp    time.Time
}

func NewSystemClockFromReply(data map[string]string) *SystemClock {
	return &SystemClock{
		Time:         data["time"],
		Date:         data["date"],
		TimeZoneName: data["time-zone-name"],
		DSTActive:    pBool(data["dst-active"]),
		Timestamp:    time.Now(),
	}
}

func (c *SystemClock) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"time":           c.Time,
		"date":           c.Date,
		"time_zone_name": c.TimeZoneName,
		"dst_active":     fmt.Sprintf("%t", c.DSTActive),
		"timestamp":      ts.Format(time.RFC3339),
	}
}

type SystemHealth struct {
	Voltage         string
	Temperature     string
	CPUTemperature  string
	PowerConsumption string
	BoardTemperature string
	Timestamp       time.Time
}

func NewSystemHealthFromReply(data map[string]string) *SystemHealth {
	return &SystemHealth{
		Voltage:          data["voltage"],
		Temperature:      data["temperature"],
		CPUTemperature:   data["cpu-temperature"],
		PowerConsumption: data["power-consumption"],
		BoardTemperature: data["board-temperature"],
		Timestamp:        time.Now(),
	}
}

func (h *SystemHealth) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"voltage":            h.Voltage,
		"temperature":        h.Temperature,
		"cpu_temperature":    h.CPUTemperature,
		"power_consumption":  h.PowerConsumption,
		"board_temperature":  h.BoardTemperature,
		"timestamp":          ts.Format(time.RFC3339),
	}
}

type SystemRouterboard struct {
	Model            string
	SerialNumber     string
	FirmwareType     string
	Firmware         string
	Revision         string
	UpgradeFirmware  string
	Routerboard      bool
	Timestamp        time.Time
}

func NewSystemRouterboardFromReply(data map[string]string) *SystemRouterboard {
	return &SystemRouterboard{
		Model:           data["model"],
		SerialNumber:    data["serial-number"],
		FirmwareType:    data["firmware-type"],
		Firmware:        data["firmware"],
		Revision:        data["revision"],
		UpgradeFirmware: data["upgrade-firmware"],
		Routerboard:     pBool(data["routerboard"]),
		Timestamp:       time.Now(),
	}
}

func (r *SystemRouterboard) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"model":            r.Model,
		"serial_number":    r.SerialNumber,
		"firmware_type":    r.FirmwareType,
		"firmware":         r.Firmware,
		"revision":         r.Revision,
		"upgrade_firmware": r.UpgradeFirmware,
		"routerboard":      fmt.Sprintf("%t", r.Routerboard),
		"timestamp":        ts.Format(time.RFC3339),
	}
}
