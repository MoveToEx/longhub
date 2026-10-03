package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"long/internal/sqlc"
	"slices"
	"strings"
	"text/template"
	"time"
)

const MaxTemplateSize = 64 * 1024
const MaxPayloadSize = 256 * 1024

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type ValueChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type Changes map[string]ValueChange

type Data struct {
	ImageID   int64       `json:"imageID"`
	Text      string      `json:"text"`
	Rating    sqlc.Rating `json:"rating"`
	Tags      []string    `json:"tags"`
	ImageURL  string      `json:"imageURL"`
	CreatedAt time.Time   `json:"createdAt"`
	Changes   *Changes    `json:"changes,omitempty"`
	Uploader  *User       `json:"uploader,omitempty"`
}

type Payload struct {
	Event     string `json:"event"`
	Data      Data   `json:"data"`
	Initiator *User  `json:"initiator,omitempty"`
}

// Embed Data so templates can use both .Data.Text and .Text.
type templateFields struct {
	Event string
	Data
	Initiator *User
}

func Compare(old, current Data) *Changes {
	changes := Changes{}
	if old.Text != current.Text {
		changes["text"] = ValueChange{Old: old.Text, New: current.Text}
	}
	if old.Rating != current.Rating {
		changes["rating"] = ValueChange{Old: old.Rating, New: current.Rating}
	}
	oldTags := sortedTags(old.Tags)
	newTags := sortedTags(current.Tags)
	if !slices.Equal(oldTags, newTags) {
		changes["tags"] = ValueChange{Old: oldTags, New: newTags}
	}
	return &changes
}

func sortedTags(tags []string) []string {
	result := append([]string{}, tags...)
	slices.Sort(result)
	return result
}

func Parse(bodyTemplate string) (*template.Template, error) {
	if len(bodyTemplate) > MaxTemplateSize {
		return nil, errors.New("body template exceeds 64 KiB")
	}
	return template.New("webhook").Funcs(template.FuncMap{
		"json": func(value any) (string, error) {
			data, err := json.Marshal(value)
			return string(data), err
		},
	}).Parse(bodyTemplate)
}

type payloadBuffer struct {
	buffer bytes.Buffer
}

func (b *payloadBuffer) Write(data []byte) (int, error) {
	if len(data) > MaxPayloadSize-b.buffer.Len() {
		return 0, errors.New("rendered body exceeds 256 KiB")
	}
	return b.buffer.Write(data)
}

func Render(bodyTemplate, event string, data Data, initiator *User) ([]byte, error) {
	if event != "create" && event != "update" && event != "delete" {
		return nil, errors.New("invalid webhook event")
	}
	if len(bodyTemplate) > MaxTemplateSize {
		return nil, errors.New("body template exceeds 64 KiB")
	}
	if strings.TrimSpace(bodyTemplate) == "" {
		return json.Marshal(Payload{Event: event, Data: data, Initiator: initiator})
	}
	tmpl, err := Parse(bodyTemplate)
	if err != nil {
		return nil, fmt.Errorf("invalid body template: %w", err)
	}
	var output payloadBuffer
	if err := tmpl.Execute(&output, templateFields{Event: event, Data: data, Initiator: initiator}); err != nil {
		return nil, fmt.Errorf("failed to render body template: %w", err)
	}
	if !json.Valid(output.buffer.Bytes()) {
		return nil, errors.New("body template must render valid JSON")
	}
	return output.buffer.Bytes(), nil
}

func PreviewData(event string) Data {
	data := Data{
		ImageID:   42,
		Text:      "Updated image caption",
		Rating:    sqlc.RatingNone,
		Tags:      []string{"landscape", "featured"},
		ImageURL:  "https://example.com/images/42.jpg",
		CreatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Uploader:  PreviewInitiator("create"),
	}
	if event == "update" {
		old := data
		old.Text = "Original image caption"
		old.Rating = sqlc.RatingModerate
		old.Tags = []string{"landscape"}
		data.Changes = Compare(old, data)
	}
	return data
}

func PreviewInitiator(event string) *User {
	if event == "create" {
		return &User{ID: 7, Username: "image-owner"}
	}
	return &User{ID: 9, Username: "image-editor"}
}
