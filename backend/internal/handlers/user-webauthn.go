package handlers

import (
	"context"
	"encoding/base64"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"
	"long/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgtype"
)

func BeginAddPasskey(c *gin.Context) {
	userID := c.GetInt64("UserID")
	w := config.GetWebAuthn()

	ctx := context.Background()

	user, err := db.Query().GetUser(ctx, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting user: %v", err)
		return
	}

	wu, err := utils.FromUser(ctx, user)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting credential: %v", err)
		return
	}

	opts, session, err := w.BeginRegistration(
		wu,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		}),
	)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when creating session")
		return
	}

	sid, _ := utils.RandomBase64String(32)

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

type ValidateAddPasskeyResponse struct {
	ID []byte `json:"id"`
}

func ValidateAddPasskey(c *gin.Context) {
	userID := c.GetInt64("UserID")
	w := config.GetWebAuthn()

	ctx := context.Background()

	user, err := db.Query().GetUser(ctx, userID)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting user")
		return
	}

	sid, err := c.Cookie("sid")
	if err != nil {
		utils.ErrorResponse(c, 400, "SID missing")
		return
	}

	session, err := db.Query().GetWebAuthnSession(ctx, sid)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when getting session")
		return
	}

	wu, err := utils.FromUser(ctx, user)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting credential: %v", err)
		return
	}

	cred, err := w.FinishRegistration(*wu, session.Data, c.Request)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when generating credential: %s", err.Error())
		return
	}

	db.Query().CreateCredential(ctx, sqlc.CreateCredentialParams{
		ID:         cred.ID,
		UserID:     userID,
		PublicKey:  cred.PublicKey,
		SignCount:  int64(cred.Authenticator.SignCount),
		Transports: textArray(cred.Transport),
		Flags:      int16(cred.Flags.ProtocolValue()),
		Aaguid: pgtype.UUID{
			Valid: true,
			Bytes: [16]byte(cred.Authenticator.AAGUID),
		},
	})

	utils.SuccessResponse(c, ValidateAddPasskeyResponse{
		ID: cred.ID,
	})
}

func DeletePasskey(c *gin.Context) {
	userID := c.GetInt64("UserID")
	ctx := context.Background()

	kid := c.Param("id")

	if len(kid) == 0 {
		utils.ErrorResponse(c, 400, "Invalid key ID")
		return
	}

	var bid []byte
	var err error

	if bid, err = base64.URLEncoding.DecodeString(kid); err != nil {
		utils.ErrorResponse(c, 400, "Invalid key ID: %v", err)
		return
	}

	err = db.Query().DeletePasskey(ctx, sqlc.DeletePasskeyParams{
		UserID: userID,
		ID:     bid,
	})

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when deleting key")
		return
	}

	utils.SuccessResponse(c, nil)
}

type EditPasskeyPayload struct {
	Name string `json:"name"`
}

func EditPasskey(c *gin.Context) {
	userID := c.GetInt64("UserID")
	ctx := context.Background()

	kid := c.Param("id")

	if len(kid) == 0 {
		utils.ErrorResponse(c, 400, "Invalid key ID")
		return
	}

	bid, err := base64.URLEncoding.DecodeString(kid)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid key ID")
		return
	}

	var payload EditPasskeyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request")
		return
	}

	err = db.Query().SetPasskeyName(ctx, sqlc.SetPasskeyNameParams{
		Name:   payload.Name,
		ID:     bid,
		UserID: userID,
	})
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when updating key")
		return
	}

	utils.SuccessResponse(c, nil)
}

func textArray(a []protocol.AuthenticatorTransport) []string {
	out := make([]string, len(a))
	for i, v := range a {
		out[i] = string(v)
	}
	return out
}

func GetPasskey(c *gin.Context) {
	userID := c.GetInt64("UserID")

	ctx := context.Background()

	keys, err := db.Query().GetPasskey(ctx, userID)

	if err != nil {
		utils.ErrorResponse(c, 500, "Failed when collecting keys")
		return
	}

	utils.SuccessResponse(c, keys)
}
