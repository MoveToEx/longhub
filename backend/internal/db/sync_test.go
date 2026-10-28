package db_test

import (
	"context"
	"errors"
	"long/internal/sqlc"
	"long/internal/testutil"
	"testing"
	"time"
)

func TestSyncVersionTracksCatalogChanges(t *testing.T) {
	pool, transact := testutil.Database(t)
	ctx := context.Background()
	q := sqlc.New(pool)
	assertVersion := func(want time.Time) {
		t.Helper()
		// Reading the version must leave it unchanged.
		for range 2 {
			got, err := q.GetSyncVersion(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if want.IsZero() {
				if got.Valid {
					t.Fatalf("empty catalog version = %v, want null", got)
				}
			} else if !got.Valid || !got.Time.Equal(want) {
				t.Fatalf("catalog version = %v, want %v", got, want)
			}
		}
	}
	versionTime := func(id int64) time.Time {
		t.Helper()
		var timestamp time.Time
		if err := pool.QueryRow(ctx, "SELECT created_at FROM version WHERE id = $1", id).Scan(&timestamp); err != nil {
			t.Fatal(err)
		}
		return timestamp
	}
	deletionTime := func() time.Time {
		t.Helper()
		var timestamp time.Time
		if err := pool.QueryRow(ctx, "SELECT deleted_at FROM image WHERE id=1").Scan(&timestamp); err != nil {
			t.Fatal(err)
		}
		return timestamp
	}

	assertVersion(time.Time{})
	testutil.Seed(t, pool)
	assertVersion(versionTime(1))

	// Publishing another upload advances the version, even without any edits.
	uploadID, err := q.InitializeVersion(ctx, sqlc.InitializeVersionParams{ImageID: 2, UserID: 2, Text: "new upload", Rating: sqlc.RatingNone})
	if err != nil {
		t.Fatal(err)
	}
	assertVersion(versionTime(uploadID))

	editID, err := q.CreateNewVersion(ctx, sqlc.CreateNewVersionParams{ImageID: 1, UserID: 2, Text: "edited text", Rating: sqlc.RatingNone, TagNames: []string{"example"}})
	if err != nil {
		t.Fatal(err)
	}
	latest := versionTime(editID)
	assertVersion(latest)

	// Uncommitted edits must never become visible to sync clients.
	rollback := errors.New("rollback")
	err = transact(ctx, func(tx *sqlc.Queries) error {
		if _, err := tx.CreateNewVersion(ctx, sqlc.CreateNewVersionParams{ImageID: 1, UserID: 2, Text: "rolled back", Rating: sqlc.RatingNone}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	assertVersion(latest)

	var requestID int64
	err = transact(ctx, func(tx *sqlc.Queries) error {
		var err error
		requestID, err = tx.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: 1, UserID: 2, Reason: "test deletion"})
		if err != nil {
			return err
		}
		if err := tx.UpdateDeletionDecision(ctx, sqlc.UpdateDeletionDecisionParams{ID: requestID, Status: sqlc.DeletionStatusApproved}); err != nil {
			return err
		}
		return tx.ApproveImageDeletion(ctx, sqlc.ApproveImageDeletionParams{ImageID: 1, RequestID: requestID})
	})
	if err != nil {
		t.Fatal(err)
	}
	deletedAt := deletionTime()
	if !deletedAt.After(latest) {
		t.Fatal("deletion did not advance the version")
	}
	assertVersion(deletedAt)

	// Indexing and webhook dispatch do not change the catalog version.
	if _, err := pool.Exec(ctx, `UPDATE image SET indexed_version = 0 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	assertVersion(deletedAt)

	// A later insertion with an older timestamp cannot move the version backward.
	if _, err := pool.Exec(ctx, `INSERT INTO version(image_id, user_id, text, rating, version, created_at)
		VALUES(2, 2, 'historical import', 'none', 2, '2020-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	assertVersion(deletedAt)
}

func TestSyncVersionIncludesDeletionWithoutMetadataVersion(t *testing.T) {
	ctx := context.Background()
	pool, _ := testutil.Database(t)
	if _, err := pool.Exec(ctx, `
INSERT INTO public.user(username,email) VALUES('uploader','uploader@example.com');
INSERT INTO image(user_id,image_key,image_url,created_at,deleted_at)
VALUES(1,'unversioned','https://example.com/unversioned','2020-01-01','2026-10-27T08:30:00Z');
`); err != nil {
		t.Fatal(err)
	}
	got, err := sqlc.New(pool).GetSyncVersion(ctx)
	if err != nil || !got.Valid || got.Time.UTC().Format("2006-01-02T15:04:05Z") != "2026-10-27T08:30:00Z" {
		t.Fatal("image deletion timestamp was ignored", got, err)
	}
}
