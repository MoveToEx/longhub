// Command setup creates a local environment file from .env.template.
//
// The template contains no application secrets. This command generates the
// Ed25519 keys used by the application and writes their standard DER
// encodings as Base64 values, which is the format expected by config.Init.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultTemplatePath = ".env.template"
	defaultOutputPath   = ".env"
)

var keyNames = []string{
	"JWT_SECRET_KEY",
	"WEBHOOK_PRIVATE_KEY",
}

func main() {
	templatePath := flag.String("template", defaultTemplatePath, "path to the environment template")
	outputPath := flag.String("output", defaultOutputPath, "path for the generated environment file")
	force := flag.Bool("force", false, "overwrite an existing output file")
	flag.Parse()

	if err := generateEnv(*templatePath, *outputPath, *force); err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s from %s\n", *outputPath, *templatePath)
}

func generateEnv(templatePath, outputPath string, force bool) error {
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template %q: %w", templatePath, err)
	}

	values, err := generateKeyValues()
	if err != nil {
		return err
	}
	result, err := fillTemplate(string(template), values)
	if err != nil {
		return err
	}

	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory %q: %w", dir, err)
		}
	}
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(outputPath, flags, 0o600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("output %q already exists (use -force to overwrite)", outputPath)
	}
	if err != nil {
		return fmt.Errorf("create output %q: %w", outputPath, err)
	}

	// Environment files contain credentials. Keep them readable only by the
	// current user where the platform supports Unix-style permissions.
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("set permissions on output %q: %w", outputPath, err)
	}
	if _, err := file.WriteString(result); err != nil {
		file.Close()
		return fmt.Errorf("write output %q: %w", outputPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close output %q: %w", outputPath, err)
	}
	return nil
}

func generateKeyValues() (map[string]string, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate JWT key pair: %w", err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("encode JWT private key: %w", err)
	}

	_, webhookPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate webhook key: %w", err)
	}
	webhookDER, err := x509.MarshalPKCS8PrivateKey(webhookPrivate)
	if err != nil {
		return nil, fmt.Errorf("encode webhook private key: %w", err)
	}

	webhookPub, err := x509.MarshalPKIXPublicKey(webhookPrivate.Public())
	if err != nil {
		return nil, fmt.Errorf("encode webhook public key: %w", err)
	}

	fmt.Printf("Webhook worker public key: %s\n", base64.StdEncoding.EncodeToString(webhookPub))

	return map[string]string{
		"JWT_SECRET_KEY":      base64.StdEncoding.EncodeToString(privateDER),
		"WEBHOOK_PRIVATE_KEY": base64.StdEncoding.EncodeToString(webhookDER),
	}, nil
}

func fillTemplate(template string, values map[string]string) (string, error) {
	lines := strings.SplitAfter(template, "\n")
	found := make(map[string]bool, len(values))
	for i, line := range lines {
		lineWithoutNewline := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		for key, value := range values {
			prefix := key + "="
			if !strings.HasPrefix(strings.TrimSpace(lineWithoutNewline), prefix) {
				continue
			}
			indent := lineWithoutNewline[:len(lineWithoutNewline)-len(strings.TrimLeft(lineWithoutNewline, " \t"))]
			comment := ""
			if commentStart := strings.Index(lineWithoutNewline, " #"); commentStart >= 0 {
				comment = lineWithoutNewline[commentStart:]
			}
			newline := line[len(lineWithoutNewline):]
			lines[i] = indent + prefix + value + comment + newline
			found[key] = true
			break
		}
	}
	for _, key := range keyNames {
		if !found[key] {
			return "", fmt.Errorf("template does not define %s", key)
		}
	}
	return strings.Join(lines, ""), nil
}
