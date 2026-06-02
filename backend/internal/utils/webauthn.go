package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"long/internal/config"
	"long/internal/db"
	"long/internal/sqlc"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/valkey-io/valkey-go"
)

type WebAuthnUser struct {
	Handle      []byte
	Username    string
	Credentials []webauthn.Credential
}

func (u WebAuthnUser) WebAuthnID() []byte {
	return u.Handle
}

func (u WebAuthnUser) WebAuthnName() string {
	return u.Username
}

func (u WebAuthnUser) WebAuthnDisplayName() string {
	return u.Username
}

func (u WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func FromUser(ctx context.Context, u sqlc.User) (*WebAuthnUser, error) {
	var result []webauthn.Credential

	cred, err := db.Query().GetCredentials(ctx, u.ID)

	if err != nil {
		return nil, err
	}

	for _, it := range cred {
		result = append(result, webauthn.Credential{
			ID:        it.ID,
			PublicKey: it.PublicKey,
			Transport: parseTransport(it.Transports),
			Flags:     webauthn.NewCredentialFlags(protocol.AuthenticatorFlags(it.Flags)),
			Authenticator: webauthn.Authenticator{
				AAGUID:    it.Aaguid.Bytes[:],
				SignCount: uint32(it.SignCount),
			},
		})
	}

	return &WebAuthnUser{
		Handle:      u.Handle,
		Username:    u.Username,
		Credentials: result,
	}, nil
}

func parseTransport(a []string) []protocol.AuthenticatorTransport {
	out := make([]protocol.AuthenticatorTransport, len(a))
	for i, v := range a {
		out[i] = protocol.AuthenticatorTransport(v)
	}
	return out
}

func SaveWebAuthnSession(ctx context.Context, data *webauthn.SessionData) (string, error) {
	sid, err := RandomBase64String(32)

	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("webauthn:%s", sid)
	value, err := json.Marshal(data)

	if err != nil {
		return "", err
	}

	client := config.Valkey()

	cmd := client.B().Set().Key(key).Value(valkey.BinaryString(value)).Build()
	err = client.Do(ctx, cmd).Error()

	if err != nil {
		return "", err
	}

	return sid, nil
}

func GetWebAuthnSession(ctx context.Context, sid string) (*webauthn.SessionData, error) {
	key := fmt.Sprintf("webauthn:%s", sid)
	client := config.Valkey()

	bytes, err := client.Do(ctx, client.B().Get().Key(key).Build()).AsBytes()
	if err != nil {
		return nil, err
	}

	data := new(webauthn.SessionData)

	if err := json.Unmarshal(bytes, data); err != nil {
		return nil, err
	}

	return data, nil
}
