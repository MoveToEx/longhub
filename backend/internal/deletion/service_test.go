package deletion_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"long/internal/deletion"
	"long/internal/sqlc"
	"long/internal/testutil"
	"strings"
	"sync"
	"testing"
)

func TestTransitions(t *testing.T) {
	allowed := map[string]string{"active/ACCEPT": "approved", "active/REJECT": "rejected", "approved/REVOKE": "revoked", "rejected/REOPEN": "active", "rejected/ACCEPT": "approved", "revoked/REOPEN": "active", "revoked/ACCEPT": "approved"}
	for _, status := range []string{"active", "approved", "rejected", "revoked"} {
		for _, action := range []string{"ACCEPT", "REJECT", "REOPEN", "REVOKE"} {
			got, err := deletion.NextStatus(sqlc.DeletionStatus(status), action)
			want, ok := allowed[status+"/"+action]
			if ok && (err != nil || string(got) != want) || !ok && !errors.Is(err, deletion.ErrConflict) {
				t.Fatalf("%s/%s => %s %v", status, action, got, err)
			}
		}
	}
}

func TestReasons(t *testing.T) {
	service := deletion.Service{Transact: func(context.Context, func(*sqlc.Queries) error) error {
		t.Fatal("invalid reason touched DB")
		return nil
	}}
	for _, reason := range []string{"", " \n\t", strings.Repeat("字", 2001)} {
		if _, err := service.Create(context.Background(), 1, 1, reason); !errors.Is(err, deletion.ErrReason) {
			t.Fatal(err)
		}
	}
}

func TestLifecycleAndVisibility(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	q := sqlc.New(pool)
	service := deletion.Service{Transact: transact}
	assertVisibility := func(want int) {
		t.Helper()
		checks := map[string]func() (int, error){
			"browse": func() (int, error) { return rowCount(q.ListImages(ctx, sqlc.ListImagesParams{Limit: 25})) },
			"user list": func() (int, error) {
				return rowCount(q.GetImagesByUser(ctx, sqlc.GetImagesByUserParams{UserID: 2, Limit: 25}))
			},
			"favorite list": func() (int, error) {
				return rowCount(q.GetFavoriteImages(ctx, sqlc.GetFavoriteImagesParams{UserID: 2, Limit: 25}))
			},
			"shortcut": func() (int, error) {
				return rowCount(q.GetFavoriteByShortcut(ctx, sqlc.GetFavoriteByShortcutParams{UserID: 2, Keyword: pgtype.Text{String: "fav", Valid: true}}))
			},
			"quick search": func() (int, error) {
				return rowCount(q.QuickSearchFavorites(ctx, sqlc.QuickSearchFavoritesParams{UserID: 2, Keywords: []string{"fav"}}))
			},
			"random": func() (int, error) {
				return rowCount(q.GetRandomImage(ctx, sqlc.GetRandomImageParams{Name: "example", Limit: 25}))
			},
			"recommendations": func() (int, error) {
				return rowCount(q.GetFavoriteTags(ctx, sqlc.GetFavoriteTagsParams{UserID: 2, Limit: 25}))
			},
		}
		for name, check := range checks {
			if count, err := check(); err != nil || count != want {
				t.Fatalf("%s visibility: %d want%d %v", name, count, want, err)
			}
		}
		if total, err := q.CountImages(ctx); err != nil || total != int64(want) {
			t.Fatal("browse total", total, err)
		}
		if total, err := q.CountImagesByUser(ctx, 2); err != nil || total != int64(want) {
			t.Fatal("user total", total, err)
		}
	}
	assertVisibility(1)
	id, err := service.Create(ctx, 1, 2, "  duplicate  ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, 1, 2, "again"); !errors.Is(err, deletion.ErrConflict) {
		t.Fatal("duplicate", err)
	}
	other, err := service.Create(ctx, 1, 3, "guidelines")
	if err != nil {
		t.Fatal(err)
	}
	row, err := q.GetDeletionRequestRecord(ctx, id)
	if err != nil || row.Reason != "duplicate" {
		t.Fatal(row, err)
	}
	if _, err := service.Create(ctx, 999, 2, "missing"); !errors.Is(err, deletion.ErrNotFound) {
		t.Fatal(err)
	}
	moderate := func(action string, expected sqlc.DeletionStatus) {
		t.Helper()
		if err := service.Moderate(ctx, id, 1, action, expected); err != nil {
			t.Fatal(err)
		}
	}
	moderate("REJECT", "active")
	moderate("REOPEN", "rejected")
	row, _ = q.GetDeletionRequestRecord(ctx, id)
	if row.ProcessedAt.Valid || row.ProcessedBy.Valid {
		t.Fatal("reopen retained processing fields")
	}
	moderate("ACCEPT", "active")
	assertVisibility(0)
	row, _ = q.GetDeletionRequestRecord(ctx, id)
	if !row.ImageDeletedAt.Valid || row.ActiveDeletionID.Int64 != id || row.ProcessedBy.Int64 != 1 || row.ModeratorUsername.String != "admin" || !row.ProcessedAt.Valid {
		t.Fatal(row)
	}
	if err := service.Moderate(ctx, other, 1, "ACCEPT", "active"); !errors.Is(err, deletion.ErrConflict) {
		t.Fatal("two approvals", err)
	}
	if err := service.Moderate(ctx, id, 1, "REJECT", "active"); !errors.Is(err, deletion.ErrConflict) {
		t.Fatal("stale", err)
	}
	if _, err := service.Create(ctx, 1, 3, "deleted"); !errors.Is(err, deletion.ErrConflict) {
		t.Fatal(err)
	}
	if _, err := q.GetImage(ctx, 1); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("write query exposes deleted image", err)
	}
	detail, err := q.GetImageDetails(ctx, 1)
	if err != nil || detail.Text != "original text" || detail.DeletionReason != "duplicate" {
		t.Fatal(detail, err)
	}
	if versions, err := q.GetImageVersions(ctx, 1); err != nil || len(versions) != 1 {
		t.Fatal("version history", err)
	}
	if rows, err := q.GetImagesByIds(ctx, []int64{1}); err != nil || len(rows) != 0 {
		t.Fatal("search leaked", err)
	}
	if total, err := q.CountFavoriteImages(ctx, 2); err != nil || total != 0 {
		t.Fatal("favorite total", total, err)
	}
	if n, err := q.AddFavorite(ctx, sqlc.AddFavoriteParams{UserID: 3, ID: 1}); err != nil || n != 0 {
		t.Fatal("new favorite allowed", err)
	}
	var payload []byte
	if err := pool.QueryRow(ctx, "SELECT payload FROM deletion_outbox WHERE kind='delete'").Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var event deletion.Event
	if err := json.Unmarshal(payload, &event); err != nil || event.Initiator.ID != 1 || event.Initiator.Username != "admin" || event.Data.Text != "original text" {
		t.Fatal(event, err)
	}
	moderate("REVOKE", "approved")
	assertVisibility(1)
	if total, err := q.CountFavoriteImages(ctx, 2); err != nil || total != 1 {
		t.Fatal("restoration lost favorite", err)
	}
	if rows, err := q.GetImagesByIds(ctx, []int64{1}); err != nil || len(rows) != 1 {
		t.Fatal("restoration missing", err)
	}
	moderate("ACCEPT", "revoked")
	moderate("REVOKE", "approved")
	moderate("REOPEN", "revoked")
	var deletes, searches int
	if err := pool.QueryRow(ctx, "SELECT count(*) FILTER(WHERE kind='delete'),count(*) FILTER(WHERE kind='search') FROM deletion_outbox").Scan(&deletes, &searches); err != nil || deletes != 2 || searches != 4 {
		t.Fatal(deletes, searches, err)
	}
	empty, err := service.Create(ctx, 2, 2, strings.Repeat("字", 2000))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Moderate(ctx, empty, 1, "ACCEPT", "active"); err != nil {
		t.Fatal("unversioned approval", err)
	}
	if detail, err := q.GetImageDetails(ctx, 2); err != nil || detail.Version != 0 || !detail.DeletedAt.Valid {
		t.Fatal(detail, err)
	}
}

func rowCount[T any](rows []T, err error) (int, error) { return len(rows), err }

func TestConcurrentDecisionsAndRollback(t *testing.T) {
	pool, transact := testutil.Database(t)
	testutil.Seed(t, pool)
	ctx := context.Background()
	q := sqlc.New(pool)
	service := deletion.Service{Transact: transact}
	first, _ := service.Create(ctx, 1, 2, "first")
	second, _ := service.Create(ctx, 1, 3, "second")
	run := func(ids []int64, action string, status sqlc.DeletionStatus) int64 {
		t.Helper()
		var wg sync.WaitGroup
		results := make(chan struct {
			id  int64
			err error
		}, len(ids))
		for _, id := range ids {
			wg.Go(func() {
				results <- struct {
					id  int64
					err error
				}{id, service.Moderate(ctx, id, 1, action, status)}
			})
		}
		wg.Wait()
		close(results)
		var winner int64
		success := 0
		for result := range results {
			if result.err == nil {
				success++
				winner = result.id
			} else if !errors.Is(result.err, deletion.ErrConflict) {
				t.Fatal(result.err)
			}
		}
		if success != 1 {
			t.Fatalf("%d successful %s", success, action)
		}
		return winner
	}
	winner := run([]int64{first, second}, "ACCEPT", "active")
	run([]int64{winner, winner}, "REVOKE", "approved")
	var events int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM deletion_outbox WHERE kind='delete'").Scan(&events)
	if events != 1 {
		t.Fatal(events)
	}
	injected := errors.New("forced rollback")
	broken := deletion.Service{Transact: func(ctx context.Context, f func(*sqlc.Queries) error) error {
		return transact(ctx, func(q *sqlc.Queries) error {
			if err := f(q); err != nil {
				return err
			}
			return injected
		})
	}}
	if err := broken.Moderate(ctx, winner, 1, "ACCEPT", "revoked"); !errors.Is(err, injected) {
		t.Fatal(err)
	}
	row, _ := q.GetDeletionRequestRecord(ctx, winner)
	if row.Status != "revoked" || row.ImageDeletedAt.Valid {
		t.Fatal("rollback inconsistent", row)
	}
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM deletion_outbox WHERE kind='delete'").Scan(&events)
	if events != 1 {
		t.Fatal("rolled back event persisted")
	}
}
