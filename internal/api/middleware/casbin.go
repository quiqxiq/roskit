package middleware

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// CasbinMiddleware enforces RBAC via Casbin using (userID, tenantSlug, routePath, httpMethod).
// Requires AuthMiddleware + TenantMiddleware to have run first so userID and tenantSlug exist.
func CasbinMiddleware(e *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, ok := c.Get("userID")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"data": nil, "error": "unauthenticated"})
			return
		}
		userID, ok := userIDVal.(uint)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"data": nil, "error": "invalid user context"})
			return
		}

		tenantSlugVal, ok := c.Get("tenantSlug")
		if !ok {
			c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "missing tenant context"})
			return
		}
		tenantSlug, ok := tenantSlugVal.(string)
		if !ok {
			c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "invalid tenant context"})
			return
		}

		sub := fmt.Sprintf("%d", userID)
		obj := c.FullPath()
		act := c.Request.Method

		allowed, err := e.Enforce(sub, tenantSlug, obj, act)
		if err != nil || !allowed {
			c.AbortWithStatusJSON(403, gin.H{"data": nil, "error": "forbidden"})
			return
		}
		c.Next()
	}
}
