package middleware

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
)

// RouterLookup is the minimal interface RouterTenantMiddleware uses.
type RouterLookup interface {
	GetByID(ctx context.Context, tenantID, id uint) (*models.Router, error)
}

// RouterTenantMiddleware validates that the :routerId URL param belongs to the
// tenant resolved by TenantMiddleware. Mount this on any route group that has
// :routerId in its path so handlers using the bridge directly can trust the ID.
//
// Aborts with 404 if the router doesn't exist or belongs to a different tenant.
// Does NOT enforce permissions — that is Casbin's job.
func RouterTenantMiddleware(repo RouterLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		ridStr := c.Param("routerId")
		if ridStr == "" {
			c.Next()
			return
		}
		rid, err := strconv.ParseUint(ridStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"data": nil, "error": "invalid router id"})
			return
		}

		tenantIDVal, ok := c.Get("tenantID")
		if !ok {
			c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "tenant context required"})
			return
		}
		tenantID, ok := tenantIDVal.(uint)
		if !ok {
			c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "invalid tenant context"})
			return
		}

		router, err := repo.GetByID(c.Request.Context(), tenantID, uint(rid))
		if err != nil {
			c.AbortWithStatusJSON(404, gin.H{"data": nil, "error": "router not found"})
			return
		}
		c.Set("router", router)
		c.Next()
	}
}
