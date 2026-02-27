package config

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/joho/godotenv"
)

type JWTConfig struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	SessionTTL int
}

type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	URLPrefix       string
}

type WebAuthnConfig struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
}

type Config struct {
	DatabaseURL string
	CORSOrigin  string
	JWT         JWTConfig
	S3          S3Config
	WebAuthn    WebAuthnConfig
}

var config Config

func mustParsePrivateKey(s string) ed25519.PrivateKey {
	bytes, err := base64.StdEncoding.DecodeString(s)

	if err != nil {
		panic("Failed when decoding private key from base64")
	}

	key, err := x509.ParsePKCS8PrivateKey(bytes)

	if err != nil {
		panic("Unrecognized private key")
	}

	result, ok := key.(ed25519.PrivateKey)

	if !ok {
		panic("Private key is not an ED25519 private key")
	}

	return result
}

func mustParsePublicKey(s string) ed25519.PublicKey {
	bytes, err := base64.StdEncoding.DecodeString(s)

	if err != nil {
		panic("Failed when decoding public key from base64")
	}

	key, err := x509.ParsePKIXPublicKey(bytes)

	if err != nil {
		panic("Unrecognized public key")
	}

	result, ok := key.(ed25519.PublicKey)

	if !ok {
		panic("Private key is not an ED25519 public key")
	}

	return result
}

func LoadConfig() {
	err := godotenv.Load()

	if err != nil {
		log.Println("Failed when loading env file, will proceed with system env")
	}

	config = Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		CORSOrigin:  os.Getenv("CORS_ORIGIN"),
		JWT: JWTConfig{
			PrivateKey: mustParsePrivateKey(os.Getenv("JWT_SECRET_KEY")),
			SessionTTL: 72,
			PublicKey:  mustParsePublicKey(os.Getenv("JWT_PUBLIC_KEY")),
		},
		S3: S3Config{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			BucketName:      os.Getenv("S3_BUCKET_NAME"),
			URLPrefix:       os.Getenv("S3_URL_PREFIX"),
		},
		WebAuthn: WebAuthnConfig{
			RPDisplayName: os.Getenv("WEBAUTHN_RP_DISPLAY_NAME"),
			RPID:          os.Getenv("WEBAUTHN_RPID"),
			RPOrigins:     strings.Split(os.Getenv("WEBAUTHN_RP_ORIGINS"), ";"),
		},
	}
}

func GetConfig() *Config {
	return &config
}

var s3client *s3.Client

func InitS3() {
	cfg := aws.Config{
		Region: "us-east-1",
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(
				config.S3.AccessKeyID,
				config.S3.SecretAccessKey,
				"",
			),
		),
	}
	s3client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(config.S3.Endpoint)
	})
}

func GetS3Client() *s3.Client {
	return s3client
}

var webAuthn *webauthn.WebAuthn

func InitWebAuthn() error {
	var err error

	webAuthn, err = webauthn.New(&webauthn.Config{
		RPID:          GetConfig().WebAuthn.RPID,
		RPDisplayName: GetConfig().WebAuthn.RPDisplayName,
		RPOrigins:     GetConfig().WebAuthn.RPOrigins,
	})

	if err != nil {
		return err
	}

	return nil
}

func GetWebAuthn() *webauthn.WebAuthn {
	return webAuthn
}
