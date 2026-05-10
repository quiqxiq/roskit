package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
)

func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"data": nil, "error": "forbidden"})
			return
		}
		role, ok := roleVal.(models.UserRole)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"data": nil, "error": "forbidden"})
			return
		}
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"data": nil, "error": "forbidden"})
	}
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.UserRoleAdmin)
}

// RouterOwnershipMiddleware resolves the :routerId URL param, verifies the router
// exists, and stores routerID (uint) in the gin context. In single-instance mode
// all authenticated users may access all routers, so no tenant scope is needed.
func RouterOwnershipMiddleware(routerRepo repository.RouterRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := c.Param("routerId")
		if param == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"data": nil, "error": "missing routerId"})
			return
		}
		id, err := strconv.ParseUint(param, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid routerId"})
			return
		}
		router, err := routerRepo.GetByID(c.Request.Context(), uint(id))
		if err != nil || router == nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
			return
		}
		c.Set("routerID", router.ID)
		c.Next()
	}
}
