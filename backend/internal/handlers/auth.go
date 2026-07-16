package handlers

import (
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
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

	ctx := c.Request.Context()

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

	ctx := c.Request.Context()

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

func BeginWebAuthnLogin(c *gin.Context) {
	w := config.GetWebAuthn()

	opts, session, err := w.BeginDiscoverableMediatedLogin(protocol.MediationDefault)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating auth session")
		return
	}

	ctx := c.Request.Context()
	sid, err := utils.SaveWebAuthnSession(ctx, session)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating session")
		return
	}

	c.SetCookie("sid", sid, 30000, "/", "", false, true)

	utils.SuccessResponse(c, *opts)
}

func ValidateWebAuthnLogin(c *gin.Context) {
	w := config.GetWebAuthn()
	ctx := c.Request.Context()
	sid, err := c.Cookie("sid")
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	session, err := utils.GetWebAuthnSession(ctx, sid)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid session")
		return
	}

	var resolved sqlc.User

	_, err = w.FinishDiscoverableLogin(func(_, handle []byte) (webauthn.User, error) {
		var err error

		resolved, err = db.Query().GetUserByHandle(ctx, handle)
		if err != nil {
			return nil, err
		}

		user, err := utils.FromUser(ctx, resolved)
		if err != nil {
			return nil, err
		}

		return user, nil
	}, *session, c.Request)

	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid credential")
		return
	}

	token, err := utils.NewToken(resolved.ID, resolved.Permission)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when generating token: %v", err)
		return
	}

	utils.SuccessResponse(c, LoginResponse{
		Token: token,
	})
}

type GetIdentityResponse struct {
	AuthorizedVia string `json:"authorizedVia"`
	sqlc.User
}

func GetIdentity(c *gin.Context) {
	userID := c.GetInt64("UserID")
	authorization := c.GetString("AuthorizedVia")

	ctx := c.Request.Context()

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
