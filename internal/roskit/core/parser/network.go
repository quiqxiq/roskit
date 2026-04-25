package parser

import (
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/core/model"
)

func init() {
	RegisterStreamParser("interface", parseInterface)
	RegisterStreamParser("arp", parseARP)
	RegisterStreamParser("dhcp_lease", parseDHCPLease)
	RegisterStreamParser("ip_pool", parseIPPool)
	RegisterStreamParser("firewall_nat", parseFirewallNAT)
	RegisterStreamParser("queue_simple", parseQueueSimple)
}

func parseInterface(_ string, pairs map[string]string) *ParseResult {
	i := model.NewInterfaceFromReply(pairs)
	return &ParseResult{
		EntityID:  i.ID,
		CacheData: i.ToCacheData(time.Now()),
	}
}

func parseARP(_ string, pairs map[string]string) *ParseResult {
	a := model.NewARPEntryFromReply(pairs)
	return &ParseResult{
		EntityID:  a.ID,
		CacheData: a.ToCacheData(time.Now()),
	}
}

func parseDHCPLease(_ string, pairs map[string]string) *ParseResult {
	d := model.NewDHCPLeaseFromReply(pairs)
	return &ParseResult{
		EntityID:  d.ID,
		CacheData: d.ToCacheData(time.Now()),
	}
}

func parseIPPool(_ string, pairs map[string]string) *ParseResult {
	p := model.NewIPPoolFromReply(pairs)
	return &ParseResult{
		EntityID:  p.ID,
		CacheData: p.ToCacheData(time.Now()),
	}
}

func parseFirewallNAT(_ string, pairs map[string]string) *ParseResult {
	n := model.NewFirewallNATFromReply(pairs)
	return &ParseResult{
		EntityID:  n.ID,
		CacheData: n.ToCacheData(time.Now()),
	}
}

func parseQueueSimple(_ string, pairs map[string]string) *ParseResult {
	q := model.NewQueueSimpleFromReply(pairs)
	return &ParseResult{
		EntityID:  q.ID,
		CacheData: q.ToCacheData(time.Now()),
	}
}
