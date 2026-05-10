package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/quiqxiq/roskit/internal/services"
)

func AuthMiddleware(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		if qp := c.Query("token"); qp != "" {
			tokenStr = qp
		} else {
			header := c.GetHeader("Authorization")
			if header == "" {
				c.AbortWithStatusJSON(401, gin.H{"data": nil, "error": "missing authorization header"})
				return
			}
			var ok bool
			tokenStr, ok = strings.CutPrefix(header, "Bearer ")
			if !ok {
				c.AbortWithStatusJSON(401, gin.H{"data": nil, "error": "invalid authorization format"})
				return
			}
		}

		claims, err := authSvc.ValidateToken(c.Request.Context(), tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"data": nil, "error": "invalid or expired token"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("tokenID", claims.TokenID)
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func LoggerMiddleware() gin.HandlerFunc {
	logger := slog.Default()
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()[:8]
		c.Set("requestID", requestID)

		c.Next()

		latency := time.Since(start)
		userID, _ := c.Get("userID")

		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_id", userID,
			"request_id", requestID,
		)
	}
}
