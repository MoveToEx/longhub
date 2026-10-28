package admin

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type graphQLTestResult struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message    string         `json:"message"`
		Extensions map[string]any `json:"extensions"`
	} `json:"errors"`
}

// Middleware permission loading has its own tests; this supplies its output to the real HTTP GraphQL handler.
func invokeGraphQL(t *testing.T, actor, permission int64, query string, variables any) graphQLTestResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/admin/graphql", func(c *gin.Context) { c.Set("UserID", actor); c.Set("Permission", permission) }, GraphQL)
	request := httptest.NewRequest("POST", "/admin/graphql", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 200 && response.Code != 422 {
		t.Fatal(response.Code, response.Body.String())
	}
	var result graphQLTestResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err, response.Body.String())
	}
	return result
}
