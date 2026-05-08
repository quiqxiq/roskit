package service

import (
	"context"
	"strings"
)

type SalesRecord struct {
	ID            string
	Date          string
	Time          string
	Username      string
	Price         string
	IPAddress     string
	MACAddress    string
	Validity      string
	Profile       string
	Comment       string
	Owner         string
	Source        string
	ScriptComment string
}

func ParseSalesRecordFromScript(name, owner, source, scriptComment string) *SalesRecord {
	parts := strings.SplitN(name, "-|-", 9)
	if len(parts) < 8 {
		return nil
	}

	rec := &SalesRecord{
		Date:          parts[0],
		Time:          parts[1],
		Username:      parts[2],
		Price:         parts[3],
		IPAddress:     parts[4],
		MACAddress:    parts[5],
		Validity:      parts[6],
		Profile:       parts[7],
		Owner:         owner,
		Source:        source,
		ScriptComment: scriptComment,
	}

	if len(parts) == 9 {
		rec.Comment = parts[8]
	}

	return rec
}

func (b *Bridge) FetchMonthlySales(ctx context.Context, routerID, owner string) ([]*SalesRecord, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?owner="+owner, "?comment=mikhmon")
	if err != nil {
		return nil, err
	}
	return parseSalesRecords(rows), nil
}

func (b *Bridge) ImportSalesFromRouterOS(ctx context.Context, routerID, owner string) ([]*SalesRecord, error) {
	if owner != "" {
		return b.FetchMonthlySales(ctx, routerID, owner)
	}
	rows, err := b.Query(ctx, routerID, "system/script/print", "?comment=mikhmon")
	if err != nil {
		return nil, err
	}

	var records []*SalesRecord
	for _, row := range rows {
		rec := ParseSalesRecordFromScript(
			row["name"], row["owner"], row["source"], row["comment"])
		if rec != nil && rec.Price != "0" && rec.Price != "" {
			rec.ID = row[".id"]
			records = append(records, rec)
		}
	}
	return records, nil
}

func parseSalesRecords(rows []map[string]string) []*SalesRecord {
	var records []*SalesRecord
	for _, row := range rows {
		rec := ParseSalesRecordFromScript(
			row["name"], row["owner"], row["source"], row["comment"])
		if rec != nil {
			rec.ID = row[".id"]
			records = append(records, rec)
		}
	}
	return records
}

