package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"long/internal/sqlc"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type testStore struct {
	images           []sqlc.AdminListImagesRow
	users            []sqlc.AdminListUsersRow
	imageParams      sqlc.AdminListImagesParams
	imageCountParams sqlc.AdminCountImagesParams
	userParams       sqlc.AdminListUsersParams
	userCountParams  sqlc.AdminCountUsersParams
	err              error
	calls            int
}

func (s *testStore) AdminListImages(_ context.Context, p sqlc.AdminListImagesParams) ([]sqlc.AdminListImagesRow, error) {
	s.calls++
	s.imageParams = p
	return s.images, s.err
}
func (s *testStore) AdminCountImages(_ context.Context, p sqlc.AdminCountImagesParams) (int64, error) {
	s.imageCountParams = p
	return int64(len(s.images)), s.err
}
func (s *testStore) AdminListUsers(_ context.Context, p sqlc.AdminListUsersParams) ([]sqlc.AdminListUsersRow, error) {
	s.calls++
	s.userParams = p
	return s.users, s.err
}
func (s *testStore) AdminCountUsers(_ context.Context, p sqlc.AdminCountUsersParams) (int64, error) {
	s.userCountParams = p
	return int64(len(s.users)), s.err
}

func request(t *testing.T, store Store, query string, variables any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/admin/graphql", NewHandler(store))
	req := httptest.NewRequest("POST", "/admin/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func TestAdminGraphQLData(t *testing.T) {
	now := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	stamp := pgtype.Timestamptz{Time: now, Valid: true}
	store := &testStore{
		images: []sqlc.AdminListImagesRow{
			{ID: 9007199254740993, ImageUrl: "https://example.com/image.png", ImageKey: "image.png", UserID: 1, UploaderUsername: "uploader", CreatedAt: stamp, UpdatedAt: stamp, VersionID: pgtype.Int8{Int64: 3, Valid: true}, Version: pgtype.Int4{Int32: 2, Valid: true}, Text: pgtype.Text{String: "hello", Valid: true}, Rating: sqlc.NullRating{Rating: sqlc.RatingNone, Valid: true}, Tags: []string{"tag"}},
			{ID: 2, CreatedAt: stamp, UpdatedAt: stamp, DeletedAt: stamp},
		},
		users: []sqlc.AdminListUsersRow{{ID: 9007199254740993, Username: "admin", Email: "admin@example.com", Permission: 8, CreatedAt: stamp, ActiveImageCount: 3, VersionCount: 5}},
	}
	response := request(t, store, `{ images { total items { id imageUrl imageKey createdAt updatedAt deletedAt uploader { id username } currentVersion { id version text rating tags } } } users { total items { id username email permission createdAt activeImageCount versionCount } } }`, nil)
	if response.Code != 200 || strings.Contains(response.Body.String(), `"errors"`) {
		t.Fatal(response.Body.String())
	}
	var result struct {
		Data struct {
			Images struct {
				Items []struct {
					ID             string
					CurrentVersion *struct{ Text string }
					DeletedAt      *time.Time
				}
				Total int
			}
			Users struct {
				Items []struct {
					ID, Email                                  string
					Permission, ActiveImageCount, VersionCount int
				}
			}
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Data.Images.Items[0].ID != "9007199254740993" || result.Data.Images.Items[0].CurrentVersion.Text != "hello" {
		t.Fatal(response.Body.String())
	}
	if result.Data.Images.Items[1].CurrentVersion != nil || result.Data.Images.Items[1].DeletedAt == nil {
		t.Fatal("nullable version/deletion was lost")
	}
	user := result.Data.Users.Items[0]
	if user.Email != "admin@example.com" || user.Permission != 8 || user.ActiveImageCount != 3 || user.VersionCount != 5 {
		t.Fatal(response.Body.String())
	}
	if store.imageParams.PageLimit != 25 || store.imageParams.Status != "ACTIVE" || store.imageParams.SortBy != "CREATED_AT" || store.imageParams.Direction != "DESC" {
		t.Fatalf("unexpected defaults: %+v", store.imageParams)
	}
	if store.userParams.PageLimit != 25 {
		t.Fatal("missing default user limit")
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("admin responses must not be cached")
	}
}

func TestAdminQueryVariables(t *testing.T) {
	store := &testStore{}
	response := request(t, store, `query($images: ImageQueryInput, $users: UserQueryInput) { images(input: $images) { items { id } total } users(input: $users) { items { id } total } }`, map[string]any{
		"images": map[string]any{"search": "  42  ", "status": "ALL", "offset": 50, "limit": 50, "sortBy": "UPDATED_AT", "direction": "ASC"},
		"users":  map[string]any{"search": "a_%", "sortBy": "EMAIL", "direction": "ASC"},
	})
	if response.Code != 200 || strings.Contains(response.Body.String(), `"errors"`) {
		t.Fatal(response.Body.String())
	}
	if store.imageParams.SearchID != 42 || store.imageParams.Search != "42" || store.imageParams.Status != "ALL" || store.imageParams.PageOffset != 50 {
		t.Fatalf("incorrect image input: %+v", store.imageParams)
	}
	if store.imageCountParams.Search != store.imageParams.Search || store.imageCountParams.Status != store.imageParams.Status {
		t.Fatal("list/count filter mismatch")
	}
	if store.userParams.Pattern != `%a\_\%%` || store.userParams.SortBy != "EMAIL" || store.userParams.PageLimit != 25 {
		t.Fatalf("incorrect user input: %+v", store.userParams)
	}
	if store.userCountParams.Pattern != store.userParams.Pattern {
		t.Fatal("list/count user search mismatch")
	}
	if !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatal("empty results must serialize as arrays")
	}
}

func TestAdminQueryRejectsInvalidInputAndSecrets(t *testing.T) {
	for _, query := range []string{
		`{ images(input: {offset: -1}) { total } }`,
		`{ users(input: {limit: 0}) { total } }`,
		`{ images(input: {limit: 101}) { total } }`,
		`{ users(input: {sortBy: PASSWORD_HASH}) { total } }`,
		`{ users { items { passwordHash handle preference } } }`,
		`mutation { deleteUser(id: "1") }`,
		`{ images(input: {search: "` + strings.Repeat("a", 256) + `"}) { total } }`,
	} {
		store := &testStore{}
		response := request(t, store, query, nil)
		if !strings.Contains(response.Body.String(), `"errors"`) || store.calls != 0 {
			t.Fatalf("query should fail before DB calls: %s: %s", query, response.Body)
		}
	}
}

func TestAdminQueryFailureIsSanitized(t *testing.T) {
	response := request(t, &testStore{err: errors.New("postgres password=private")}, `{ users { total } }`, nil)
	if !strings.Contains(response.Body.String(), "Unable to load admin data") || strings.Contains(response.Body.String(), "password") {
		t.Fatal(response.Body.String())
	}
}

func TestAdminQueryLimits(t *testing.T) {
	var query strings.Builder
	query.WriteString("{")
	for i := range 60 {
		query.WriteString("u" + strconv.Itoa(i) + ": users { total } ")
	}
	query.WriteString("}")
	store := &testStore{}
	response := request(t, store, query.String(), nil)
	if !strings.Contains(response.Body.String(), "complexity") || store.calls != 0 {
		t.Fatal(response.Body.String())
	}
	router := gin.New()
	router.POST("/admin/graphql", NewHandler(store))
	req := httptest.NewRequest("POST", "/admin/graphql", strings.NewReader(strings.Repeat(" ", maxRequestBytes+1)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != 413 {
		t.Fatalf("oversize body status: %d", recorder.Code)
	}
}
