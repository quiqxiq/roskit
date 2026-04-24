package parser

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
)

func init() {
	RegisterStreamParser("system_scheduler", parseSystemScheduler)
	RegisterStreamParser("system_script", parseSystemScript)

	RegisterPollParser("system_identity", parseSystemIdentity)
	RegisterPollParser("system_clock", parseSystemClock)
	RegisterPollParser("system_health", parseSystemHealth)
	RegisterPollParser("system_routerboard", parseSystemRouterboard)
}

func parseSystemScheduler(_ string, pairs map[string]string) *ParseResult {
	s := model.NewSystemSchedulerFromReply(pairs)
	return &ParseResult{
		EntityID:  s.ID,
		CacheData: s.ToCacheData(time.Now()),
	}
}

func parseSystemScript(_ string, pairs map[string]string) *ParseResult {
	s := model.NewSystemScriptFromReply(pairs)
	return &ParseResult{
		EntityID:  s.ID,
		CacheData: s.ToCacheData(time.Now()),
	}
}

func parseSystemIdentity(_ string, rows []map[string]string) []*ParseResult {
	if len(rows) == 0 {
		return nil
	}
	i := model.NewSystemIdentityFromReply(rows[0])
	return []*ParseResult{{
		EntityID:  "singleton",
		CacheData: i.ToCacheData(time.Now()),
	}}
}

func parseSystemClock(_ string, rows []map[string]string) []*ParseResult {
	if len(rows) == 0 {
		return nil
	}
	c := model.NewSystemClockFromReply(rows[0])
	return []*ParseResult{{
		EntityID:  "singleton",
		CacheData: c.ToCacheData(time.Now()),
	}}
}

func parseSystemHealth(_ string, rows []map[string]string) []*ParseResult {
	if len(rows) == 0 {
		return nil
	}
	h := model.NewSystemHealthFromReply(rows[0])
	return []*ParseResult{{
		EntityID:  "singleton",
		CacheData: h.ToCacheData(time.Now()),
	}}
}

func parseSystemRouterboard(_ string, rows []map[string]string) []*ParseResult {
	if len(rows) == 0 {
		return nil
	}
	r := model.NewSystemRouterboardFromReply(rows[0])
	return []*ParseResult{{
		EntityID:  "singleton",
		CacheData: r.ToCacheData(time.Now()),
	}}
}
