package casbinx

import (
	"embed"
	"fmt"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"

	"github.com/quiqxiq/roskit/internal/models"
)

//go:embed model.conf
var modelFS embed.FS

func NewEnforcer(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("casbin adapter: %w", err)
	}

	src, err := modelFS.ReadFile("model.conf")
	if err != nil {
		return nil, fmt.Errorf("read casbin model: %w", err)
	}
	m, err := model.NewModelFromString(string(src))
	if err != nil {
		return nil, fmt.Errorf("casbin model: %w", err)
	}

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("casbin enforcer: %w", err)
	}

	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("load casbin policy: %w", err)
	}
	return e, nil
}

// SeedPolicies menyimpan role policies ke DB. Idempotent — aman dipanggil ulang.
func SeedPolicies(e *casbin.Enforcer) error {
	for _, p := range getRolePolicies() {
		has, err := e.HasPolicy(p[0], p[1], p[2], p[3])
		if err != nil {
			return fmt.Errorf("has policy: %w", err)
		}
		if has {
			continue
		}
		if _, err := e.AddPolicy(p[0], p[1], p[2], p[3]); err != nil {
			return fmt.Errorf("add policy %v: %w", p, err)
		}
	}
	return nil
}

// AssignRole memberikan role ke user dalam satu tenant.
// Untuk superadmin, tenantSlug = models.PlatformTenantSlug ("__platform__").
func AssignRole(e *casbin.Enforcer, userID uint, tenantSlug string, role models.UserRole) error {
	sub := fmt.Sprintf("%d", userID)
	for _, r := range e.GetRolesForUserInDomain(sub, tenantSlug) {
		if r == string(role) {
			return nil
		}
	}
	if _, err := e.AddRoleForUserInDomain(sub, string(role), tenantSlug); err != nil {
		return fmt.Errorf("add role: %w", err)
	}
	return nil
}

// RevokeRoles mencabut semua role user dalam satu tenant.
func RevokeRoles(e *casbin.Enforcer, userID uint, tenantSlug string) error {
	sub := fmt.Sprintf("%d", userID)
	if _, err := e.DeleteRolesForUserInDomain(sub, tenantSlug); err != nil {
		return fmt.Errorf("delete roles: %w", err)
	}
	return nil
}
