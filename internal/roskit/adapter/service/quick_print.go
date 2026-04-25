package service

import (
	"context"
	"fmt"
	"time"
)

const QuickPrintComment = "QuickPrintMikhmon"

type QuickPrintPackage struct {
	ID           string
	Name         string
	Server       string
	UserMode     string
	UserLength   string
	Prefix       string
	CharMode     string
	Profile      string
	TimeLimit    string
	DataLimit    string
	Comment      string
	Validity     string
	Price        string
	SellingPrice string
	LockUser     string
	Timestamp    time.Time
}

func (q *QuickPrintPackage) ToSource() string {
	return "#" + q.Name +
		"#" + q.Server +
		"#" + q.UserMode +
		"#" + q.UserLength +
		"#" + q.Prefix +
		"#" + q.CharMode +
		"#" + q.Profile +
		"#" + q.TimeLimit +
		"#" + q.DataLimit +
		"#" + q.Comment +
		"#" + q.Validity +
		"#" + q.Price + "_" + q.SellingPrice +
		"#" + q.LockUser
}

func ParseQuickPrintSource(source string) *QuickPrintPackage {
	parts := splitQuickPrint(source)
	if len(parts) < 12 {
		return nil
	}

	price, sellingPrice := splitPrice(parts[12])

	return &QuickPrintPackage{
		Name:         parts[1],
		Server:       parts[2],
		UserMode:     parts[3],
		UserLength:   parts[4],
		Prefix:       parts[5],
		CharMode:     parts[6],
		Profile:      parts[7],
		TimeLimit:    parts[8],
		DataLimit:    parts[9],
		Comment:      parts[10],
		Validity:     parts[11],
		Price:        price,
		SellingPrice: sellingPrice,
		LockUser:     fieldOrEmpty(parts, 13),
	}
}

func splitQuickPrint(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '#' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func splitPrice(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func fieldOrEmpty(parts []string, idx int) string {
	if idx < len(parts) {
		return parts[idx]
	}
	return ""
}

func (b *Bridge) SaveQuickPrintPackage(ctx context.Context, routerID string, pkg *QuickPrintPackage) error {
	_, err := b.mutateAdd(ctx, routerID, "system/script/add", map[string]string{
		"name":    pkg.Name,
		"source":  pkg.ToSource(),
		"comment": QuickPrintComment,
	})
	return err
}

func (b *Bridge) UpdateQuickPrintPackage(ctx context.Context, routerID, id string, pkg *QuickPrintPackage) error {
	args := []string{"=.id=" + id, "=name=" + pkg.Name, "=source=" + pkg.ToSource(), "=comment=" + QuickPrintComment}
	_, err := b.Mutate(ctx, routerID, "system/script/set", args...)
	return err
}

func (b *Bridge) RemoveQuickPrintPackage(ctx context.Context, routerID, id string) error {
	_, err := b.Mutate(ctx, routerID, "system/script/remove", "=.id="+id)
	return err
}

func (b *Bridge) GetQuickPrintPackage(ctx context.Context, routerID, name string) (*QuickPrintPackage, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?name="+name, "?comment="+QuickPrintComment)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("quick print package '%s' not found", name)
	}

	pkg := ParseQuickPrintSource(rows[0]["source"])
	if pkg == nil {
		return nil, fmt.Errorf("failed to parse quick print source")
	}
	pkg.ID = rows[0][".id"]
	return pkg, nil
}

func (b *Bridge) ListQuickPrintPackages(ctx context.Context, routerID string) ([]*QuickPrintPackage, error) {
	rows, err := b.Query(ctx, routerID, "system/script/print",
		"?comment="+QuickPrintComment)
	if err != nil {
		return nil, err
	}

	var packages []*QuickPrintPackage
	for _, row := range rows {
		pkg := ParseQuickPrintSource(row["source"])
		if pkg != nil {
			pkg.ID = row[".id"]
			packages = append(packages, pkg)
		}
	}
	return packages, nil
}

var _ = time.Second
