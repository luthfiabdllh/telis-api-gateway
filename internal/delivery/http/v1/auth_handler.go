package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"telis-api-gateway/internal/domain"
)

type AuthHandler struct {
	authUsecase domain.AuthUsecase
	ssoSecret   string
}

func NewAuthHandler(public *gin.RouterGroup, protected *gin.RouterGroup, authUsecase domain.AuthUsecase, ssoSecret string) {
	handler := &AuthHandler{
		authUsecase: authUsecase,
		ssoSecret:   ssoSecret,
	}

	authRoutes := public.Group("/auth")
	{
		authRoutes.POST("/login", handler.Login)
		authRoutes.POST("/sso/google", handler.LoginSSO)
		authRoutes.POST("/refresh", handler.Refresh)
	}

	protectedAuthRoutes := protected.Group("/auth")
	{
		protectedAuthRoutes.POST("/register", handler.Register)
	}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	RoleID   int    `json:"role_id" binding:"required"`
}

// Register godoc
// @Summary Daftarkan pengguna baru (Khusus Admin)
// @Description Mendaftarkan pengguna baru dengan Role tertentu. Hanya user dengan Role Admin yang bisa mengeksekusi ini.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body v1.RegisterRequest true "Payload Pendaftaran"
// @Success 201 {object} map[string]interface{} "Berhasil didaftarkan"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (Bukan Admin)"
// @Failure 409 {object} map[string]interface{} "Conflict (Email/Username sudah dipakai)"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role.(string) != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can register new users"})
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authUsecase.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.RoleID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login godoc
// @Summary Login pengguna
// @Description Menghasilkan JWT Access Token dan Refresh Token jika kredensial valid.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body v1.LoginRequest true "Payload Login"
// @Success 200 {object} map[string]interface{} "Berhasil login, mengembalikan token"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized (Password salah atau user tidak ada)"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.authUsecase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh godoc
// @Summary Refresh Token
// @Description Memperbarui Access Token menggunakan Refresh Token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body v1.RefreshRequest true "Payload Refresh"
// @Success 200 {object} map[string]interface{} "Berhasil refresh, mengembalikan token baru"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized (Token tidak valid)"
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newAccessToken, newRefreshToken, err := h.authUsecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
	})
}

type SSOLoginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// LoginSSO godoc
// @Summary Login SSO (Internal Frontend Only)
// @Description Login menggunakan email dari provider SSO (seperti Google). Memerlukan header X-Internal-Secret.
// @Tags Auth
// @Accept json
// @Produce json
// @Param X-Internal-Secret header string true "Internal Secret"
// @Param request body v1.SSOLoginRequest true "Payload SSO Login"
// @Success 200 {object} map[string]interface{} "Berhasil login, mengembalikan token"
// @Failure 401 {object} map[string]interface{} "Unauthorized (Secret salah atau Akun tidak terdaftar)"
// @Router /auth/sso/google [post]
func (h *AuthHandler) LoginSSO(c *gin.Context) {
	internalSecret := c.GetHeader("X-Internal-Secret")
	if internalSecret != h.ssoSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid internal secret"})
		return
	}

	var req SSOLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.authUsecase.LoginSSO(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"role":          "User", // we can ignore passing exact role here since jwt payload will have it
	})
}
