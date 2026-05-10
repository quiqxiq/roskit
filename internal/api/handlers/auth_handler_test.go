package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/quiqxiq/roskit/internal/api/handlers"
	apitesthelpers "github.com/quiqxiq/roskit/internal/api/testhelpers"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newAuthService(repo services.UserRepository) *services.AuthService {
	return services.NewAuthService(repo, nil, []byte("test-secret"), []byte("refresh-secret"))
}

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{
		User: &models.User{
			ID:           1,
			Username:     "admin",
			PasswordHash: hashedPassword(t, "password123"),
			Role:         models.UserRoleAdmin,
			Active:       true,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc, nil)

	router := gin.New()
	router.POST("/login", h.Login)

	body := `{"username":"admin","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{
		User: &models.User{
			ID:           1,
			Username:     "admin",
			PasswordHash: hashedPassword(t, "correct"),
			Role:         models.UserRoleAdmin,
			Active:       true,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc, nil)

	router := gin.New()
	router.POST("/login", h.Login)

	body := `{"username":"admin","password":"wrong"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_MissingBody(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc, nil)

	router := gin.New()
	router.POST("/login", h.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_InactiveUser(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{
		User: &models.User{
			ID:           2,
			Username:     "disabled",
			PasswordHash: hashedPassword(t, "pass"),
			Role:         models.UserRoleAdmin,
			Active:       false,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc, nil)

	router := gin.New()
	router.POST("/login", h.Login)

	body := `{"username":"disabled","password":"pass"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthHandler_Me(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc, nil)

	router := gin.New()
	router.GET("/me", func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("username", "admin")
		c.Set("role", models.UserRoleAdmin)
		h.Me(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "admin", data["role"])
}
