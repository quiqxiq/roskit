package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
)

// TenantLookup is the minimal interface TenantMiddleware needs to resolve a tenant slug.
type TenantLookup interface {
	GetBySlug(ctx context.Context, slug string) (*models.Tenant, error)
}

// TenantMiddleware resolves tenant context from JWT claims, with superadmin override
// via the X-Tenant-Slug header.
//
// Behavior:
//   - Superadmin: takes tenant slug from X-Tenant-Slug header; empty = "__platform__"
//     (platform-level ops such as /admin/*).
//   - Non-superadmin: tenant slug is fixed from the JWT and cannot be overridden.
//   - Any non-platform slug resolved must point to an active (not suspended) tenant.
//
// Sets into context:
//   - tenantSlug (string): always present
//   - tenantID (uint): only when tenantSlug != "__platform__"
//   - tenant (*models.Tenant): only when tenantSlug != "__platform__"
func TenantMiddleware(repo TenantLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, ok := c.Get("role")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"data": nil, "error": "unauthenticated"})
			return
		}
		role, _ := roleVal.(models.UserRole)

		var tenantSlug string
		if role == models.UserRoleSuperAdmin {
			tenantSlug = c.GetHeader("X-Tenant-Slug")
			if tenantSlug == "" {
				tenantSlug = models.PlatformTenantSlug
			}
		} else {
			slug, _ := c.Get("jwtTenantSlug")
			tenantSlug, _ = slug.(string)
			if tenantSlug == "" || tenantSlug == models.PlatformTenantSlug {
				c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "missing tenant context"})
				return
			}
		}

		if tenantSlug != models.PlatformTenantSlug {
			tenant, err := repo.GetBySlug(c.Request.Context(), tenantSlug)
			if err != nil {
				c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "tenant not accessible"})
				return
			}
			if tenant.Status == models.TenantStatusSuspended {
				c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "tenant suspended"})
				return
			}
			// For non-superadmin, reject attempt to access a different tenant than the JWT.
			if role != models.UserRoleSuperAdmin {
				jwtTID, _ := c.Get("jwtTenantID")
				if tid, ok := jwtTID.(uint); ok && tid != tenant.ID {
					c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "tenant mismatch"})
					return
				}
			}
			c.Set("tenant", tenant)
			c.Set("tenantID", tenant.ID)
		}
		c.Set("tenantSlug", tenantSlug)
		c.Next()
	}
}
