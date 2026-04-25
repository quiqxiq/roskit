package service

import (
	"context"
	"fmt"
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

func (b *Bridge) RecordSale(ctx context.Context, routerID string, rec *SalesRecord) error {
	name := fmt.Sprintf("%s-|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|-%s",
		rec.Date, rec.Time, rec.Username, rec.Price,
		rec.IPAddress, rec.MACAddress, rec.Validity, rec.Profile, rec.Comment)

	_, err := b.mutateAdd(ctx, routerID, "system/script/add", map[string]string{
		"name":    name,
		"owner":   rec.Owner,
		"source":  rec.Source,
		"comment": "mikhmon",
	})
	return err
}

func (b *Bridge) RemoveSaleRecord(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/script/remove", "=.id="+id)
	return err
}

func (b *Bridge) RemoveAllSalesForMonth(ctx context.Context, routerID, owner string) error {
	records, err := b.FetchMonthlySales(ctx, routerID, owner)
	if err != nil {
		return err
	}
	for _, rec := range records {
		if rec.ID != "" {
			if err := b.RemoveSaleRecord(ctx, routerID, rec.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Bridge) FetchDailySales(ctx context.Context, routerID, date string) ([]*SalesRecord, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?source="+date, "?comment=mikhmon")
	if err != nil {
		return nil, err
	}
	return parseSalesRecords(rows), nil
}

func (b *Bridge) FetchDailySalesCount(ctx context.Context, routerID, date string) (int, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?source="+date, "?comment=mikhmon")
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (b *Bridge) FetchMonthlySales(ctx context.Context, routerID, owner string) ([]*SalesRecord, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?owner="+owner, "?comment=mikhmon")
	if err != nil {
		return nil, err
	}
	return parseSalesRecords(rows), nil
}

func (b *Bridge) FetchMonthlySalesCount(ctx context.Context, routerID, owner string) (int, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?owner="+owner, "?comment=mikhmon")
	if err != nil {
		return 0, err
	}
	return len(rows), nil
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
