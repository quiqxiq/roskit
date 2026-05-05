package services

import (
	"context"
	"fmt"

	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

type UserStatus struct {
	Active    bool   `json:"active"`
	Username  string `json:"username"`
	Profile   string `json:"profile"`
	IPAddress string `json:"ip_address"`
	MAC       string `json:"mac"`
	Uptime    string `json:"uptime"`
	BytesIn   string `json:"bytes_in"`
	BytesOut  string `json:"bytes_out"`
	TimeLeft  string `json:"time_left"`
	Comment   string `json:"comment"`
}

type StatusService struct {
	routerRepo *repository.RouterRepo
	bridge     *roskitservice.Bridge
}

func NewStatusService(routerRepo *repository.RouterRepo, bridge *roskitservice.Bridge) *StatusService {
	return &StatusService{
		routerRepo: routerRepo,
		bridge:     bridge,
	}
}

func (s *StatusService) GetUserStatus(ctx context.Context, sessionName string, mac string) (*UserStatus, error) {
	router, err := s.routerRepo.GetBySessionName(ctx, sessionName)
	if err != nil {
		return nil, fmt.Errorf("router not found: %w", err)
	}
	rID := fmt.Sprintf("%d", router.ID)

	active, err := s.bridge.Query(ctx, rID, "ip/hotspot/active/print", "?mac-address="+mac)
	if err != nil {
		return nil, err
	}

	if len(active) > 0 {
		session := active[0]
		return &UserStatus{
			Active:    true,
			Username:  session["user"],
			Profile:   session["profile"],
			IPAddress: session["address"],
			MAC:       mac,
			Uptime:    session["uptime"],
			BytesIn:   session["bytes-in"],
			BytesOut:  session["bytes-out"],
			TimeLeft:  session["session-time-left"],
			Comment:   session["comment"],
		}, nil
	}

	return &UserStatus{
		Active: false,
		MAC:    mac,
	}, nil
}
