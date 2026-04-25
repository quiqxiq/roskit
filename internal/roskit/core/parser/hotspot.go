package parser

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/core/model"
)

type ParseResult struct {
	CacheData map[string]string
	IsDead    bool
	EntityID  string
}

var streamParsers = map[string]StreamParseFunc{}

type StreamParseFunc func(routerID string, pairs map[string]string) *ParseResult

func RegisterStreamParser(measurement string, fn StreamParseFunc) {
	streamParsers[measurement] = fn
}

func ParseStreamSentence(routerID string, meta *command.CommandMeta, pairs map[string]string) *ParseResult {
	if len(pairs) == 0 {
		return nil
	}
	if IsDead(pairs) {
		id := pairs[".id"]
		if id == "" {
			return nil
		}
		return &ParseResult{
			IsDead:   true,
			EntityID: id,
			CacheData: map[string]string{
				".id": id,
			},
		}
	}
	fn, ok := streamParsers[meta.Measurement]
	if !ok {
		return parseGeneric(pairs)
	}
	return fn(routerID, pairs)
}

func parseGeneric(pairs map[string]string) *ParseResult {
	id := pairs[".id"]
	if id == "" {
		return nil
	}
	return &ParseResult{
		EntityID:  id,
		CacheData: pairs,
		IsDead:    false,
	}
}

var pollParsers = map[string]PollParseFunc{}

type PollParseFunc func(routerID string, rows []map[string]string) []*ParseResult

func RegisterPollParser(measurement string, fn PollParseFunc) {
	pollParsers[measurement] = fn
}

func ParsePollReply(routerID string, meta *command.CommandMeta, rows []map[string]string) []*ParseResult {
	fn, ok := pollParsers[meta.Measurement]
	if !ok {
		return parseGenericPoll(rows)
	}
	return fn(routerID, rows)
}

func parseGenericPoll(rows []map[string]string) []*ParseResult {
	var results []*ParseResult
	for _, row := range rows {
		id := row[".id"]
		if id == "" {
			id = "singleton"
		}
		results = append(results, &ParseResult{
			EntityID:  id,
			CacheData: row,
		})
	}
	return results
}

func init() {
	RegisterStreamParser("hotspot_user", parseHotspotUser)
	RegisterStreamParser("hotspot_active", parseHotspotActive)
	RegisterStreamParser("hotspot_profile", parseHotspotProfile)
	RegisterStreamParser("hotspot_server", parseHotspotServer)
	RegisterStreamParser("hotspot_host", parseHotspotHost)
	RegisterStreamParser("hotspot_cookie", parseHotspotCookie)
	RegisterStreamParser("ip_binding", parseIPBinding)
	RegisterStreamParser("walled_garden", parseWalledGarden)

	RegisterPollParser("system_resource", parseSystemResource)
}

func parseHotspotUser(_ string, pairs map[string]string) *ParseResult {
	u := model.NewHotspotUserFromReply(pairs)
	return &ParseResult{
		EntityID:  u.ID,
		CacheData: u.ToCacheData(time.Now()),
	}
}

func parseHotspotActive(_ string, pairs map[string]string) *ParseResult {
	a := model.NewHotspotActiveFromReply(pairs)
	return &ParseResult{
		EntityID:  a.ID,
		CacheData: a.ToCacheData(time.Now()),
	}
}

func parseHotspotProfile(_ string, pairs map[string]string) *ParseResult {
	p := model.NewHotspotProfileFromReply(pairs)
	return &ParseResult{
		EntityID:  p.ID,
		CacheData: p.ToCacheData(time.Now()),
	}
}

func parseHotspotServer(_ string, pairs map[string]string) *ParseResult {
	s := &model.HotspotServer{
		ID:        pairs[".id"],
		Name:      pairs["name"],
		Interface: pairs["interface"],
		Disabled:  ParseBool(pairs["disabled"]),
		Timestamp: time.Now(),
	}
	return &ParseResult{
		EntityID:  s.ID,
		CacheData: s.ToCacheData(time.Now()),
	}
}

func parseHotspotHost(_ string, pairs map[string]string) *ParseResult {
	h := &model.HotspotHost{
		ID:         pairs[".id"],
		MACAddress: pairs["mac-address"],
		Address:    pairs["address"],
		Server:     pairs["server"],
		ToAddress:  pairs["to-address"],
		Authorized: ParseBool(pairs["authorized"]),
		Timestamp:  time.Now(),
	}
	return &ParseResult{
		EntityID:  h.ID,
		CacheData: h.ToCacheData(time.Now()),
	}
}

func parseHotspotCookie(_ string, pairs map[string]string) *ParseResult {
	c := &model.HotspotCookie{
		ID:        pairs[".id"],
		User:      pairs["user"],
		MAC:       pairs["mac-address"],
		Address:   pairs["address"],
		Timestamp: time.Now(),
	}
	return &ParseResult{
		EntityID:  c.ID,
		CacheData: c.ToCacheData(time.Now()),
	}
}

func parseIPBinding(_ string, pairs map[string]string) *ParseResult {
	b := &model.IPBinding{
		ID:        pairs[".id"],
		MAC:       pairs["mac-address"],
		Address:   pairs["address"],
		Type:      pairs["type"],
		Disabled:  ParseBool(pairs["disabled"]),
		Comment:   pairs["comment"],
		Timestamp: time.Now(),
	}
	return &ParseResult{
		EntityID:  b.ID,
		CacheData: b.ToCacheData(time.Now()),
	}
}

func parseSystemResource(_ string, rows []map[string]string) []*ParseResult {
	if len(rows) == 0 {
		return nil
	}
	r := model.NewSystemResourceFromReply(rows[0])
	return []*ParseResult{{
		EntityID:  "singleton",
		CacheData: r.ToCacheData(time.Now()),
	}}
}

func parseWalledGarden(_ string, pairs map[string]string) *ParseResult {
	w := model.NewHotspotWalledGardenFromReply(pairs)
	return &ParseResult{
		EntityID:  w.ID,
		CacheData: w.ToCacheData(time.Now()),
	}
}
