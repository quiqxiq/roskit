package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/url"

	"github.com/quiqxiq/roskit/internal/models"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

type TemplateRepository interface {
	Create(ctx context.Context, t *models.PrintTemplate) error
	GetByID(ctx context.Context, id uint) (*models.PrintTemplate, error)
	List(ctx context.Context, routerID uint) ([]models.PrintTemplate, error)
	Update(ctx context.Context, t *models.PrintTemplate) error
	Delete(ctx context.Context, id uint) error
	GetByRouterAndType(ctx context.Context, routerID uint, templateType string) ([]models.PrintTemplate, error)
}

// VoucherTemplateVars holds all variables available inside a print template.
// Template content uses Go html/template syntax, e.g. {{.Username}}.
type VoucherTemplateVars struct {
	Num         int
	Username    string
	Password    string
	Validity    string
	TimeLimit   string
	DataLimit   string
	Price       string
	Profile     string
	Comment     string
	HotspotName string
	DNSName     string
	Logo        string
	UserMode    string       // "vc" (username=password) or "up" (separate username & password)
	QR          string       // "yes" or "no"
	QRCode      template.HTML // rendered <img> tag for QR code
}

// RenderedVoucher is the rendered HTML for a single voucher.
type RenderedVoucher struct {
	Num  int    `json:"num"`
	HTML string `json:"html"`
}

// RenderParams carries per-batch context needed for rendering.
// Profile-level fields (Validity, TimeLimit, etc.) are shared across all vouchers in one print batch.
type RenderParams struct {
	HotspotName string
	DNSName     string
	Logo        string
	UserMode    string // "vc" or "up"
	Currency    string
	Profile     string
	Validity    string
	TimeLimit   string
	DataLimit   string
	Price       string
	Comment     string
}

type CreateTemplateRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Part    string `json:"part" binding:"required,oneof=header row footer"`
	Content string `json:"content" binding:"required"`
}

type UpdateTemplateRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Part    string `json:"part" binding:"omitempty,oneof=header row footer"`
	Content string `json:"content"`
}

type TemplateService struct {
	repo   TemplateRepository
	logger *slog.Logger
}

func NewTemplateService(repo TemplateRepository) *TemplateService {
	return &TemplateService{
		repo:   repo,
		logger: slog.Default().With("component", "template-svc"),
	}
}

func (s *TemplateService) Create(ctx context.Context, routerID uint, req CreateTemplateRequest) (*models.PrintTemplate, error) {
	t := &models.PrintTemplate{
		Name:     req.Name,
		Type:     req.Type,
		Part:     req.Part,
		Content:  req.Content,
		RouterID: routerID,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return t, nil
}

func (s *TemplateService) GetByID(ctx context.Context, id uint) (*models.PrintTemplate, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TemplateService) List(ctx context.Context, routerID uint) ([]models.PrintTemplate, error) {
	return s.repo.List(ctx, routerID)
}

func (s *TemplateService) Update(ctx context.Context, id uint, req UpdateTemplateRequest) (*models.PrintTemplate, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		t.Name = req.Name
	}
	if req.Type != "" {
		t.Type = req.Type
	}
	if req.Part != "" {
		t.Part = req.Part
	}
	if req.Content != "" {
		t.Content = req.Content
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return t, nil
}

func (s *TemplateService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// Render renders a named template type for the given vouchers.
// Templates are split into header/row/footer parts; each voucher gets header+row+footer assembled.
func (s *TemplateService) Render(
	ctx context.Context,
	routerID uint,
	templateType string,
	vouchers []roskitservice.GeneratedVoucher,
	params RenderParams,
) ([]RenderedVoucher, error) {
	parts, err := s.repo.GetByRouterAndType(ctx, routerID, templateType)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no template found for type %q on router %d", templateType, routerID)
	}

	byPart := map[string]string{}
	for _, p := range parts {
		byPart[p.Part] = p.Content
	}

	results := make([]RenderedVoucher, 0, len(vouchers))
	for i, v := range vouchers {
		vars := buildVars(i+1, v, params)

		var buf bytes.Buffer
		for _, partName := range []string{"header", "row", "footer"} {
			content, ok := byPart[partName]
			if !ok {
				continue
			}
			rendered, err := renderPart(content, vars)
			if err != nil {
				s.logger.Warn("template render error", "part", partName, "voucher", v.Username, "error", err)
				buf.WriteString(content)
			} else {
				buf.WriteString(rendered)
			}
		}

		results = append(results, RenderedVoucher{Num: i + 1, HTML: buf.String()})
	}

	return results, nil
}

func buildVars(num int, v roskitservice.GeneratedVoucher, p RenderParams) VoucherTemplateVars {
	userMode := p.UserMode
	if userMode == "" {
		userMode = "vc"
	}

	qr := "no"
	var qrCode template.HTML
	if p.DNSName != "" {
		qr = "yes"
		loginURL := "http://" + p.DNSName
		qrURL := "https://api.qrserver.com/v1/create-qr-code/?size=80x80&data=" + url.QueryEscape(loginURL)
		qrCode = template.HTML(`<img src="` + qrURL + `" class="qrcode" alt="QR">`)
	}

	return VoucherTemplateVars{
		Num:         num,
		Username:    v.Username,
		Password:    v.Password,
		Validity:    p.Validity,
		TimeLimit:   p.TimeLimit,
		DataLimit:   p.DataLimit,
		Price:       p.Price,
		Profile:     p.Profile,
		Comment:     p.Comment,
		HotspotName: p.HotspotName,
		DNSName:     p.DNSName,
		Logo:        p.Logo,
		UserMode:    userMode,
		QR:          qr,
		QRCode:      qrCode,
	}
}

func renderPart(content string, vars VoucherTemplateVars) (string, error) {
	tmpl, err := template.New("").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
