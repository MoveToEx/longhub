package webhook

import (
	"errors"
	"fmt"
	"net/textproto"
	"strings"
)

const MaxHeaders = 32
const MaxHeaderValueSize = 8 * 1024
const MaxHeadersSize = 64 * 1024

// Header names are fixed; each value is a plain text Go template using the same
// fields and functions as a body template.
func RenderHeaders(templates map[string]string, event string, data Data, initiator *User) (map[string]string, error) {
	if event != "create" && event != "update" && event != "delete" {
		return nil, errors.New("invalid webhook event")
	}
	if len(templates) > MaxHeaders {
		return nil, errors.New("at most 32 custom headers are allowed")
	}
	headers := make(map[string]string, len(templates))
	sourceSize, renderedSize := 0, 0
	for name, source := range templates {
		if !validHeaderName(name) {
			return nil, fmt.Errorf("invalid header name %q", name)
		}
		name = textproto.CanonicalMIMEHeaderKey(name)
		if reservedHeader(name) {
			return nil, fmt.Errorf("header %q is reserved", name)
		}
		if _, exists := headers[name]; exists {
			return nil, fmt.Errorf("duplicate header %q (names are case-insensitive)", name)
		}
		sourceSize += len(name) + len(source)
		if sourceSize > MaxHeadersSize {
			return nil, errors.New("header templates exceed 64 KiB")
		}
		tmpl, err := parseTemplate(source)
		if err != nil {
			return nil, fmt.Errorf("invalid template for header %q: %w", name, err)
		}
		output := templateBuffer{maxSize: MaxHeaderValueSize, limitError: "rendered header value exceeds 8 KiB"}
		if err := tmpl.Execute(&output, templateFields{Event: event, Data: data, Initiator: initiator}); err != nil {
			return nil, fmt.Errorf("failed to render header %q: %w", name, err)
		}
		value := output.buffer.String()
		for _, c := range value {
			if c != '\t' && (c < ' ' || c > '~') {
				return nil, fmt.Errorf("header %q must contain only ASCII text without control characters", name)
			}
		}
		renderedSize += len(name) + len(value)
		if renderedSize > MaxHeadersSize {
			return nil, errors.New("rendered headers exceed 64 KiB")
		}
		headers[name] = strings.Trim(value, " \t")
	}
	return headers, nil
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c) {
			continue
		}
		return false
	}
	return true
}

func reservedHeader(name string) bool {
	switch name {
	case "X-Signature", "X-Server-Signature", "Host", "Content-Length", "Connection", "Keep-Alive", "Te", "Trailer", "Transfer-Encoding", "Upgrade":
		return true
	}
	return strings.HasPrefix(name, "Proxy-")
}
