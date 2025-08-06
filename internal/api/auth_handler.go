package api

import (
	"time"

	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewAuthHandler(services *service.Services) *AuthHandler {
	return &AuthHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Login handles user authentication (placeholder implementation)
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if !h.response.BindAndValidate(c, &req) {
		return
	}

	// Placeholder authentication logic
	// In a real implementation, you would:
	// 1. Hash the password and compare with stored hash
	// 2. Check user existence in database
	// 3. Generate JWT token with proper claims
	// 4. Set appropriate token expiration

	// For demo purposes, accept specific credentials
	if req.Username == "admin" && req.Password == "admin123" {
		// Generate a placeholder token (in real implementation, use proper JWT)
		token := "demo-jwt-token-" + time.Now().Format("20060102150405")
		expiresAt := time.Now().Add(24 * time.Hour)

		response := LoginResponse{
			Token:     token,
			ExpiresAt: expiresAt,
			User: UserInfo{
				ID:       1,
				Username: "admin",
				Role:     "admin",
			},
		}

		h.response.Success(c, response, "Login successful")
	} else {
		h.response.Unauthorized(c, "Invalid username or password")
	}
}

// Logout handles user logout (placeholder implementation)
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a real implementation, you might:
	// 1. Invalidate the token (add to blacklist)
	// 2. Clear any server-side sessions
	// 3. Log the logout event

	h.response.Success(c, nil, "Logout successful")
}

// RefreshToken refreshes an existing token (placeholder implementation)
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// In a real implementation, you would:
	// 1. Validate the existing token
	// 2. Check if it's eligible for refresh
	// 3. Generate a new token with extended expiration
	// 4. Return the new token

	// For demo purposes
	token := "refreshed-demo-jwt-token-" + time.Now().Format("20060102150405")
	expiresAt := time.Now().Add(24 * time.Hour)

	response := gin.H{
		"token":      token,
		"expires_at": expiresAt,
	}

	h.response.Success(c, response, "Token refreshed successfully")
}

// GetProfile returns the current user's profile (placeholder implementation)
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// In a real implementation, you would:
	// 1. Extract user information from JWT token
	// 2. Fetch additional user details from database
	// 3. Return user profile information

	// For demo purposes, return mock user info
	userInfo := UserInfo{
		ID:       1,
		Username: "admin",
		Role:     "admin",
	}

	h.response.Success(c, userInfo, "Profile retrieved successfully")
}