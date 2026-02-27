package handlers

import (
	"context"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type RegisterPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func LoginRoute(c *gin.Context) {
	var payload LoginPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Unable to bind parameters")
		return
	}

	ctx := context.Background()

	user, err := db.Query().GetUserByName(ctx, payload.Username)

	if err != nil {
		utils.ErrorResponse(c, 401, "Invalid credential")
		return
	}

	if user.PasswordHash == nil {
		utils.ErrorResponse(c, 400, "Invalid credential")
		return
	}

	if bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(payload.Password)) != nil {
		utils.ErrorResponse(c, 401, "Invalid credential")
		return
	}

	token, err := utils.NewToken(user.ID, user.Permission)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when generating token: %v", err)
		return
	}

	utils.SuccessResponse(c, LoginResponse{
		Token: token,
	})
}

func RegisterRoute(c *gin.Context) {
	var payload RegisterPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Unable to bind parameters")
		return
	}

	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)

	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to hash password")
		return
	}

	if _, err := db.Query().GetUserByName(ctx, payload.Username); err == nil {
		utils.ErrorResponse(c, 409, "User already exists")
		return
	}

	err = db.Query().CreateUser(ctx, sqlc.CreateUserParams{
		Email:        payload.Email,
		Username:     payload.Username,
		Permission:   db.PermissionEdit | db.PermissionCreate,
		PasswordHash: hash,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Unable to create user: %v", err)
		return
	}

	c.Status(204)
}

type GetIdentityResponse struct {
	AuthorizedVia string `json:"authorizedVia"`
	sqlc.User
}

func GetIdentity(c *gin.Context) {
	userID := c.GetInt64("UserID")
	authorization := c.GetString("AuthorizedVia")

	ctx := context.Background()

	user, err := db.Query().GetUser(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting user: %v", err)
		return
	}

	utils.SuccessResponse(c, GetIdentityResponse{
		AuthorizedVia: authorization,
		User:          user,
	})
}
