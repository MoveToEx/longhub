package webhook

import (
	"encoding/json"
	"long/internal/sqlc"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultPayload(t *testing.T) {
	for _, event := range []string{"create", "update", "delete"} {
		t.Run(event, func(t *testing.T) {
			data := PreviewData(event)
			body, err := Render("", event, data, PreviewInitiator(event))
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Event     string                     `json:"event"`
				Data      map[string]json.RawMessage `json:"data"`
				Initiator *User                      `json:"initiator"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatal(err)
			}
			if result.Event != event {
				t.Fatalf("event = %q, want %q", result.Event, event)
			}
			_, hasChanges := result.Data["changes"]
			if hasChanges != (event == "update") {
				t.Fatalf("changes present = %v for %s", hasChanges, event)
			}
			if !reflect.DeepEqual(result.Initiator, PreviewInitiator(event)) {
				t.Fatalf("unexpected initiator: %+v", result.Initiator)
			}
			for _, field := range []string{"imageID", "text", "rating", "tags", "imageURL", "createdAt", "uploader"} {
				if _, ok := result.Data[field]; !ok {
					t.Errorf("missing field %s", field)
				}
			}
		})
	}
}

func TestRenderTemplateEscapesValues(t *testing.T) {
	data := PreviewData("update")
	data.Text = "quotes: \"hello\"\nbackslash: \\ tab:\t <>&"
	body, err := Render(`{"event": {{json .Event}}, "id": {{.ImageID}}, "text": {{json .Data.Text}}, "rating": {{json .Rating}}, "tags": {{json .Tags}}, "url": {{json .ImageURL}}, "created": {{json .CreatedAt}}, "changes": {{json .Changes}}}`, "update", data, PreviewInitiator("update"))
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		ID      int64    `json:"id"`
		Text    string   `json:"text"`
		Tags    []string `json:"tags"`
		Changes Changes  `json:"changes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != data.ImageID || result.Text != data.Text || !reflect.DeepEqual(result.Tags, data.Tags) {
		t.Fatalf("rendered values differ: %+v", result)
	}
	if result.Changes["text"].Old != "Original image caption" {
		t.Fatalf("missing old text: %+v", result.Changes)
	}
}

func TestRenderRejectsInvalidTemplates(t *testing.T) {
	for _, bodyTemplate := range []string{
		`{{`,
		`{"text": {{json .Missing}}}`,
		`{"text": {{.Text}}}`,
		`{"text": {{unknown .Text}}}`,
		strings.Repeat(" ", MaxTemplateSize) + "{}",
	} {
		if _, err := Render(bodyTemplate, "create", PreviewData("create"), PreviewInitiator("create")); err == nil {
			t.Errorf("expected error for template %.80q", bodyTemplate)
		}
	}
	if _, err := Render("{}", "invalid", Data{}, nil); err == nil {
		t.Error("expected invalid event error")
	}
}

func TestRenderLimitsOutput(t *testing.T) {
	data := PreviewData("create")
	data.Text = strings.Repeat("x", MaxPayloadSize)
	if _, err := Render(`{{range .Tags}}{{$.Text}}{{end}}`, "create", data, PreviewInitiator("create")); err == nil || !strings.Contains(err.Error(), "256 KiB") {
		t.Fatalf("expected payload size error, got %v", err)
	}
}

func TestTemplateCanInspectOptionalChanges(t *testing.T) {
	data := PreviewData("update")
	data.Changes = &Changes{}
	body, err := Render(`{"old": {{with .Changes.text}}{{json .Old}}{{else}}null{{end}}}`, "update", data, PreviewInitiator("update"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"old": null}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestCompareChangedValuesOnly(t *testing.T) {
	old := Data{Text: "before", Rating: sqlc.RatingNone, Tags: []string{"a", "b"}}
	current := Data{Text: "after", Rating: sqlc.RatingViolent, Tags: []string{}}
	want := Changes{
		"text":   {Old: "before", New: "after"},
		"rating": {Old: sqlc.RatingNone, New: sqlc.RatingViolent},
		"tags":   {Old: []string{"a", "b"}, New: []string{}},
	}
	if got := Compare(old, current); !reflect.DeepEqual(*got, want) {
		t.Fatalf("changes = %+v, want %+v", *got, want)
	}
	current = old
	current.Tags = []string{"b", "a"}
	if got := Compare(old, current); len(*got) != 0 {
		t.Fatalf("tag order should not count as a change: %+v", *got)
	}
	current.Tags = nil
	old.Tags = []string{}
	current.Changes = Compare(old, current)
	body, err := Render("", "update", current, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"changes":{}`) {
		t.Fatalf("expected empty changes object: %s", body)
	}
}

func TestRenderUserIdentities(t *testing.T) {
	for _, event := range []string{"create", "update", "delete"} {
		t.Run(event, func(t *testing.T) {
			data := PreviewData(event)
			initiator := PreviewInitiator(event)
			for _, bodyTemplate := range []string{
				`{"uploader": {{json .Data.Uploader}}, "initiator": {{json .Initiator}}}`,
				`{"uploader": {{json .Uploader}}, "initiator": {{json .Initiator}}}`,
				`{"uploader": {"id": {{.Data.Uploader.ID}}, "username": {{json .Data.Uploader.Username}}}, "initiator": {"id": {{.Initiator.ID}}, "username": {{json .Initiator.Username}}}}`,
			} {
				body, err := Render(bodyTemplate, event, data, initiator)
				if err != nil {
					t.Fatal(err)
				}
				var result struct {
					Uploader  *User `json:"uploader"`
					Initiator *User `json:"initiator"`
				}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(result.Uploader, data.Uploader) || !reflect.DeepEqual(result.Initiator, initiator) {
					t.Fatalf("unexpected identities: %+v", result)
				}
				if (result.Uploader.ID == result.Initiator.ID) != (event == "create") {
					t.Fatalf("owner and initiator must be distinct for sample %s", event)
				}
			}
		})
	}
}

func TestRenderCanGuardLegacyMissingIdentities(t *testing.T) {
	body, err := Render(`{"uploader": {{with .Data.Uploader}}{{json .}}{{else}}null{{end}}, "initiator": {{with .Initiator}}{{json .}}{{else}}null{{end}}}`, "create", Data{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"uploader": null, "initiator": null}` {
		t.Fatalf("unexpected body: %s", body)
	}
}
