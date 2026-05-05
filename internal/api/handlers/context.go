package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// tenantIDFromCtx returns the resolved tenant ID from middleware. Aborts with 403 if missing.
// For superadmin routes that target a specific tenant, the TenantMiddleware sets tenantID
// from X-Tenant-Slug header. Platform-only routes (e.g. /admin/*) will not have tenantID set.
func tenantIDFromCtx(c *gin.Context) (uint, bool) {
	v, ok := c.Get("tenantID")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"data": nil, "error": "tenant context required"})
		return 0, false
	}
	id, ok := v.(uint)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"data": nil, "error": "invalid tenant context"})
		return 0, false
	}
	return id, true
}

// optionalTenantIDFromCtx returns the tenant ID if present, or (0, false) otherwise — without aborting.
// Use this in handlers shared between platform and tenant scopes.
func optionalTenantIDFromCtx(c *gin.Context) (uint, bool) {
	v, ok := c.Get("tenantID")
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}
