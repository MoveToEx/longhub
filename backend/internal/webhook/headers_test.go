package webhook

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRenderHeadersUsesBodyTemplateFieldsAsPlainText(t *testing.T) {
	for _, event := range []string{"create", "update", "delete", "restore"} {
		data, initiator := PreviewData(event), PreviewInitiator(event)
		headers, err := RenderHeaders(map[string]string{
			"authorization": "Bearer token",
			"x-event":       `{{.Event | printf "image.%s"}}`,
			"X-Image-ID":    `{{.Data.ImageID}}`,
			"X-Owner":       `{{.Uploader.Username}}`,
			"X-Actor":       `{{.Initiator.Username}}`,
			"X-JSON":        `{{json .Tags}}`,
			"X-Empty":       "",
		}, event, data, initiator)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"Authorization": "Bearer token", "X-Event": "image." + event, "X-Image-Id": "42",
			"X-Owner": "image-owner", "X-Actor": initiator.Username, "X-Json": `["landscape","featured"]`, "X-Empty": "",
		}
		if !reflect.DeepEqual(headers, want) {
			t.Fatalf("%s: got %#v, want %#v", event, headers, want)
		}
	}
}

func TestRenderHeadersRejectsInvalidConfigurationAndOutput(t *testing.T) {
	data := PreviewData("update")
	data.Text = "caption\r\nInjected: true"
	for _, headers := range []map[string]string{
		{"": "value"}, {"Bad Name": "value"}, {"Bad:Name": "value"}, {"X-Event\r\n": "value"},
		{"X-Signature": "override"}, {"x-server-signature": "override"}, {"HOST": "example.com"},
		{"Content-Length": "1"}, {"Transfer-Encoding": "chunked"}, {"Proxy-Authorization": "token"},
		{"X-Event": "one", "x-event": "two"}, {"X-Event": "{{"}, {"X-Event": "{{.Missing}}"},
		{"X-Text": "{{.Text}}"}, {"X-Value": "\x00"}, {"X-Value": "\x7f"}, {"X-Value": "世界"},
		{"X-Large": strings.Repeat("x", MaxHeaderValueSize+1)},
		{"X-Large": strings.Repeat("x", MaxHeadersSize)},
		{"X-Expanded": `{{range .Tags}}` + strings.Repeat("x", MaxHeaderValueSize/2+1) + `{{end}}`},
	} {
		if _, err := RenderHeaders(headers, "update", data, PreviewInitiator("update")); err == nil {
			t.Errorf("accepted invalid headers: %.100v", headers)
		}
	}
	headers := map[string]string{}
	for i := range MaxHeaders + 1 {
		headers[fmt.Sprintf("X-%d", i)] = "value"
	}
	if _, err := RenderHeaders(headers, "create", data, nil); err == nil {
		t.Fatal("accepted too many headers")
	}
	// The sources fit within the limit but expand beyond the combined output limit.
	headers = map[string]string{}
	for i := range 9 {
		headers[fmt.Sprintf("X-%d", i)] = `{{range .Tags}}` + strings.Repeat("x", 4000) + `{{end}}`
	}
	if _, err := RenderHeaders(headers, "create", data, nil); err == nil || !strings.Contains(err.Error(), "rendered headers exceed") {
		t.Fatalf("expected combined output limit, got %v", err)
	}
}
