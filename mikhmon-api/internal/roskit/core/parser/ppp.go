package parser

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
)

func init() {
	RegisterStreamParser("ppp_secret", parsePPPSecret)
	RegisterStreamParser("ppp_active", parsePPPActive)
}

func parsePPPSecret(_ string, pairs map[string]string) *ParseResult {
	s := model.NewPPPSecretFromReply(pairs)
	return &ParseResult{
		EntityID:  s.ID,
		CacheData: s.ToCacheData(time.Now()),
	}
}

func parsePPPActive(_ string, pairs map[string]string) *ParseResult {
	a := model.NewPPPActiveFromReply(pairs)
	return &ParseResult{
		EntityID:  a.ID,
		CacheData: a.ToCacheData(time.Now()),
	}
}
