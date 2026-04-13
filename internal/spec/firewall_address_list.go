package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// FirewallAddressListSpec streams firewall address-list entry changes.
// Detects when IPs are added/removed from blacklists, whitelists, DDoS lists.
type FirewallAddressListSpec struct{}

func NewFirewallAddressListSpec() *FirewallAddressListSpec { return &FirewallAddressListSpec{} }

func (s *FirewallAddressListSpec) Command() []string {
	return []string{
		"/ip/firewall/address-list/print",
		"=follow",
		"=.proplist=.id,list,address,timeout,disabled,dynamic,comment,creation-time",
	}
}

func (s *FirewallAddressListSpec) Tag() string         { return "firewall-addrlist" }
func (s *FirewallAddressListSpec) Measurement() string { return "firewall_address_list" }

func (s *FirewallAddressListSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}
	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}
	now := time.Now()

	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID: routerID, Measurement: s.Measurement(),
			Type: domain.EventDead, Timestamp: now,
			CacheKey: mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	tags := map[string]string{"list": pairs["list"], "address": pairs["address"]}
	fields := map[string]interface{}{
		"timeout":  pairs["timeout"],
		"disabled": mikrotik.ParseBool(pairs["disabled"]),
		"dynamic":  mikrotik.ParseBool(pairs["dynamic"]),
		"comment":  pairs["comment"],
	}
	cacheData := map[string]string{
		"id": id, "list": pairs["list"], "address": pairs["address"],
		"timeout": pairs["timeout"], "dynamic": pairs["dynamic"],
		"comment": pairs["comment"], "timestamp": now.Format(time.RFC3339),
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: tags, Fields: fields,
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: cacheData,
	}, nil
}
