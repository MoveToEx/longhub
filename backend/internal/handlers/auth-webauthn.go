package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func BeginWebAuthnLogin(c *gin.Context) {
	w := config.GetWebAuthn()

	opts, session, err := w.BeginDiscoverableMediatedLogin(protocol.MediationDefault)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating auth session")
		return
	}

	ctx := context.Background()
	sid, err := utils.RandomBase64String(32)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when generating random id")
		return
	}

	err = db.Query().CreateWebAuthnSession(ctx, sqlc.CreateWebAuthnSessionParams{
		ID:   sid,
		Data: *session,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating session")
		return
	}

	c.SetCookie("sid", sid, 30000, "/", "", false, true)

	utils.SuccessResponse(c, *opts)
}

func bytesToInt64(buf []byte) int64 {
	var result int64
	err := binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &result)
	if err != nil {
		return 0
	}
	return result
}

func ValidateWebAuthnLogin(c *gin.Context) {
	w := config.GetWebAuthn()
	ctx := context.Background()
	sid, err := c.Cookie("sid")
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	sess, err := db.Query().GetWebAuthnSession(ctx, sid)

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
	}, sess.Data, c.Request)

	if err != nil {
		e, ok := err.(*protocol.Error)
		if ok {
			utils.ErrorResponse(c, 400, "Invalid credential: %s %s", e.Details, e.DevInfo)
		}
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
