package db_test

import (
	"context"
	"long/internal/sqlc"
	"long/internal/testutil"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAdminSQL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, _ := testutil.Database(t)
	_, err := pool.Exec(ctx, `
INSERT INTO public.user (username, email, permission, created_at) VALUES
 ('admin', 'admin@example.com', 8, '2026-10-23T00:00:00Z'),
 ('uploader', 'uploader@example.com', 3, '2026-10-23T00:00:00Z'),
 ('quiet', 'quiet@example.com', 0, '2026-10-23T00:00:00Z');
INSERT INTO image (user_id, image_key, image_url, created_at, updated_at, deleted_at) VALUES
 (2, 'alpha', 'https://example.com/alpha.png', '2026-10-23T00:00:00Z', '2026-10-23T00:00:00Z', NULL),
 (2, 'beta', 'https://example.com/beta.png', '2026-10-23T00:00:00Z', '2026-10-23T00:00:00Z', '2026-10-23T01:00:00Z'),
 (2, 'empty', 'https://example.com/empty.png', '2026-10-23T00:00:00Z', '2026-10-23T00:00:00Z', NULL),
 (1, 'gamma', 'https://example.com/gamma.png', '2026-10-23T00:00:00Z', '2026-10-23T00:00:00Z', NULL);
INSERT INTO version (image_id, user_id, text, rating, version) VALUES
 (1, 2, 'old text', 'none', 1), (2, 2, 'beta', 'moderate', 1),
 (4, 1, 'gamma', 'violent', 1), (1, 2, 'alpha updated', 'none', 2);
UPDATE image SET current_version_id = CASE id WHEN 1 THEN 4 WHEN 2 THEN 2 WHEN 4 THEN 3 END;
INSERT INTO tag (name) VALUES ('featured'), ('literal_%');
INSERT INTO version_tag (version_id, tag_id) VALUES (4, 1), (4, 2);
`)
	if err != nil {
		t.Fatal(err)
	}
	q := sqlc.New(pool)
	for _, tc := range []struct {
		name, status, search, sort, direction string
		offset, limit                         int32
		ids                                   []int64
		total                                 int64
	}{
		{"active", "ACTIVE", "", "CREATED_AT", "DESC", 0, 25, []int64{4, 3, 1}, 3},
		{"deleted", "DELETED", "", "ID", "DESC", 0, 25, []int64{2}, 1},
		{"all", "ALL", "", "ID", "ASC", 0, 25, []int64{1, 2, 3, 4}, 4},
		{"page one", "ALL", "", "CREATED_AT", "DESC", 0, 2, []int64{4, 3}, 4},
		{"page two", "ALL", "", "CREATED_AT", "DESC", 2, 2, []int64{2, 1}, 4},
		{"version text", "ACTIVE", "UPDATED", "ID", "ASC", 0, 25, []int64{1}, 1},
		{"tag", "ACTIVE", "featured", "ID", "ASC", 0, 25, []int64{1}, 1},
		{"literal wildcard", "ALL", "_%", "ID", "ASC", 0, 25, []int64{1}, 1},
		{"uploader", "ALL", "UPLOADER", "ID", "ASC", 0, 25, []int64{1, 2, 3}, 3},
		{"empty", "ALL", "no match", "ID", "ASC", 0, 25, []int64{}, 0},
		{"exact image id", "ALL", "2", "ID", "ASC", 0, 25, []int64{2}, 1},
		{"tie ascending", "ALL", "", "UPDATED_AT", "ASC", 0, 25, []int64{4, 3, 2, 1}, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(tc.search) + "%"
			searchID, _ := strconv.ParseInt(tc.search, 10, 64)
			rows, err := q.AdminListImages(ctx, sqlc.AdminListImagesParams{Status: tc.status, Search: tc.search, SearchID: searchID, Pattern: pattern, SortBy: tc.sort, Direction: tc.direction, PageOffset: tc.offset, PageLimit: tc.limit})
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]int64, 0, len(rows))
			for _, row := range rows {
				ids = append(ids, row.ID)
				if row.ID == 3 && row.VersionID.Valid {
					t.Fatal("unversioned record must remain nullable")
				}
				if row.ID == 1 && (row.Version.Int32 != 2 || len(row.Tags) != 2) {
					t.Fatalf("wrong current metadata: %+v", row)
				}
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("IDs %v, want %v", ids, tc.ids)
			}
			count, err := q.AdminCountImages(ctx, sqlc.AdminCountImagesParams{Status: tc.status, Search: tc.search, SearchID: searchID, Pattern: pattern})
			if err != nil || count != tc.total {
				t.Fatalf("total %d, want %d: %v", count, tc.total, err)
			}
		})
	}
	for _, tc := range []struct {
		name, search, sort string
		searchID           int64
		ids                []int64
	}{
		{"username order", "", "USERNAME", 0, []int64{1, 3, 2}},
		{"email", "quiet@", "EMAIL", 0, []int64{3}},
		{"username", "UPLOADER", "ID", 0, []int64{2}},
		{"exact id", "3", "ID", 3, []int64{3}},
		{"permissions", "", "PERMISSION", 0, []int64{3, 2, 1}},
		{"empty users", "not found", "ID", 0, []int64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pattern := "%" + tc.search + "%"
			rows, err := q.AdminListUsers(ctx, sqlc.AdminListUsersParams{Search: tc.search, SearchID: tc.searchID, Pattern: pattern, SortBy: tc.sort, Direction: "ASC", PageLimit: 25})
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]int64, 0, len(rows))
			for _, row := range rows {
				ids = append(ids, row.ID)
				if row.ID == 2 && (row.ActiveImageCount != 2 || row.VersionCount != 3) {
					t.Fatalf("incorrect contribution counts: %+v", row)
				}
				if row.ID == 3 && (row.ActiveImageCount != 0 || row.VersionCount != 0) {
					t.Fatal("empty activity counts must be zero")
				}
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("IDs %v, want %v", ids, tc.ids)
			}
			count, err := q.AdminCountUsers(ctx, sqlc.AdminCountUsersParams{Search: tc.search, SearchID: tc.searchID, Pattern: pattern})
			if err != nil || count != int64(len(tc.ids)) {
				t.Fatalf("user count %d: %v", count, err)
			}
		})
	}
}
