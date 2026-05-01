package services

import (
	"context"
	"fmt"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/services/templates"
)

type defaultTemplateEntry struct {
	Name string
	Type string
	Part string
	Path string
}

var defaultTemplates = []defaultTemplateEntry{
	{Name: "Default", Type: "default", Part: "header", Path: "default/header.html"},
	{Name: "Default", Type: "default", Part: "row", Path: "default/row.html"},
	{Name: "Default", Type: "default", Part: "footer", Path: "default/footer.html"},
	{Name: "Small", Type: "small", Part: "header", Path: "small/header.html"},
	{Name: "Small", Type: "small", Part: "row", Path: "small/row.html"},
	{Name: "Small", Type: "small", Part: "footer", Path: "small/footer.html"},
	{Name: "Thermal", Type: "thermal", Part: "header", Path: "thermal/header.html"},
	{Name: "Thermal", Type: "thermal", Part: "row", Path: "thermal/row.html"},
	{Name: "Thermal", Type: "thermal", Part: "footer", Path: "thermal/footer.html"},
}

// SeedGlobalDefaults seeds default templates (RouterID=0) from embedded files.
// When force=false, existing templates are skipped.
// When force=true, existing global templates are updated with current embedded content.
func SeedGlobalDefaults(ctx context.Context, repo TemplateRepository, force bool) error {
	existing, err := repo.List(ctx, 0)
	if err != nil {
		return fmt.Errorf("list global templates: %w", err)
	}

	seenByKey := map[string]*models.PrintTemplate{}
	for i, t := range existing {
		if t.RouterID == 0 {
			key := t.Type + "|" + t.Part
			seenByKey[key] = &existing[i]
		}
	}

	for _, d := range defaultTemplates {
		key := d.Type + "|" + d.Part
		content, err := templates.FS.ReadFile(d.Path)
		if err != nil {
			return fmt.Errorf("read embedded template %s: %w", d.Path, err)
		}

		if tmpl, exists := seenByKey[key]; exists {
			if !force {
				continue
			}
			tmpl.Content = string(content)
			if err := repo.Update(ctx, tmpl); err != nil {
				return fmt.Errorf("update template %s/%s: %w", d.Type, d.Part, err)
			}
			continue
		}

		t := &models.PrintTemplate{
			Name:    d.Name,
			Type:    d.Type,
			Part:    d.Part,
			Content: string(content),
		}
		if err := repo.Create(ctx, t); err != nil {
			return fmt.Errorf("seed template %s/%s/%s: %w", d.Type, d.Part, d.Name, err)
		}
	}
	return nil
}
