package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	apitesthelpers "github.com/quiqxiq/roskit/internal/api/testhelpers"
	"github.com/quiqxiq/roskit/internal/api/handlers"
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
		User: &models.SystemUser{
			ID:           1,
			Username:     "admin",
			PasswordHash: hashedPassword(t, "password123"),
			Role:         "owner",
			Active:       true,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc)

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
		User: &models.SystemUser{
			ID:           1,
			Username:     "admin",
			PasswordHash: hashedPassword(t, "correct"),
			Role:         "owner",
			Active:       true,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc)

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
	h := handlers.NewAuthHandler(svc)

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
		User: &models.SystemUser{
			ID:           2,
			Username:     "disabled",
			PasswordHash: hashedPassword(t, "pass"),
			Role:         "admin",
			Active:       false,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc)

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
	h := handlers.NewAuthHandler(svc)

	router := gin.New()
	router.GET("/me", func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("username", "admin")
		c.Set("role", "owner")
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
	assert.Equal(t, "owner", data["role"])
}

func TestAuthHandler_Setup_FirstTime(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{
		UserCount: 0,
		User: &models.SystemUser{
			ID:           1,
			Username:     "owner",
			PasswordHash: hashedPassword(t, "pass1234"),
			Role:         "owner",
			Active:       true,
		},
	}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc)

	router := gin.New()
	router.POST("/setup", h.Setup)

	body := `{"username":"owner","password":"pass1234"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/setup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuthHandler_Setup_AlreadyDone(t *testing.T) {
	repo := &apitesthelpers.MockUserRepository{UserCount: 1}
	svc := newAuthService(repo)
	h := handlers.NewAuthHandler(svc)

	router := gin.New()
	router.POST("/setup", h.Setup)

	body := `{"username":"owner","password":"pass1234"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/setup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
