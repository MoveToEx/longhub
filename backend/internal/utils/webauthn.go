package utils

import (
	"context"
	"long/internal/db"
	"long/internal/sqlc"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
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
