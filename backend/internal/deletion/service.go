package deletion

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"long/internal/sqlc"
	"long/internal/webhook"
	"strings"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("image or deletion request not found")
	ErrConflict = errors.New("request or image state changed; refresh and try again")
	ErrReason   = errors.New("reason must contain between 1 and 2000 characters")
)

type Transaction func(context.Context, func(*sqlc.Queries) error) error
type Service struct{ Transact Transaction }

// Event is an immutable deletion or restoration snapshot. An omitted name is a
// legacy deletion event already stored in the outbox.
type Event struct {
	Event     string        `json:"event,omitempty"`
	Data      webhook.Data  `json:"data"`
	Initiator *webhook.User `json:"initiator"`
	Version   int32         `json:"version"`
}

func (s *Service) Create(ctx context.Context, imageID, userID int64, reason string) (int64, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 2000 {
		return 0, ErrReason
	}
	var id int64
	err := s.Transact(ctx, func(q *sqlc.Queries) error {
		image, err := q.LockImageForDeletion(ctx, imageID)
		if err != nil {
			return err
		}
		if image.DeletedAt.Valid {
			return ErrConflict
		}
		exists, err := q.HasActiveDeletionRequest(ctx, sqlc.HasActiveDeletionRequestParams{ImageID: imageID, UserID: userID})
		if err != nil {
			return err
		}
		if exists {
			return ErrConflict
		}
		id, err = q.CreateDeletionRequest(ctx, sqlc.CreateDeletionRequestParams{ImageID: imageID, UserID: userID, Reason: reason})
		return err
	})
	return id, normalizeError(err)
}

func NextStatus(status sqlc.DeletionStatus, action string) (sqlc.DeletionStatus, error) {
	switch action {
	case "ACCEPT":
		if status == sqlc.DeletionStatusActive || status == sqlc.DeletionStatusRejected || status == sqlc.DeletionStatusRevoked {
			return sqlc.DeletionStatusApproved, nil
		}
	case "REJECT":
		if status == sqlc.DeletionStatusActive {
			return sqlc.DeletionStatusRejected, nil
		}
	case "REOPEN":
		if status == sqlc.DeletionStatusRejected || status == sqlc.DeletionStatusRevoked {
			return sqlc.DeletionStatusActive, nil
		}
	case "REVOKE":
		if status == sqlc.DeletionStatusApproved {
			return sqlc.DeletionStatusRevoked, nil
		}
	}
	return "", ErrConflict
}

func (s *Service) Moderate(ctx context.Context, requestID, actorID int64, action string, expected sqlc.DeletionStatus) error {
	err := s.Transact(ctx, func(q *sqlc.Queries) error {
		imageID, err := q.GetDeletionRequestImage(ctx, requestID)
		if err != nil {
			return err
		}
		image, err := q.LockImageForDeletion(ctx, imageID)
		if err != nil {
			return err
		}
		request, err := q.LockDeletionRequest(ctx, requestID)
		if err != nil {
			return err
		}
		if request.Status != expected {
			return ErrConflict
		}
		next, err := NextStatus(request.Status, action)
		if err != nil {
			return err
		}
		if next == sqlc.DeletionStatusApproved && (image.DeletedAt.Valid || image.ActiveDeletionID.Valid) {
			return ErrConflict
		}
		if action == "REVOKE" && (!image.ActiveDeletionID.Valid || image.ActiveDeletionID.Int64 != requestID) {
			return ErrConflict
		}
		moderator := pgtype.Int8{Int64: actorID, Valid: next != sqlc.DeletionStatusActive}
		if err := q.UpdateDeletionDecision(ctx, sqlc.UpdateDeletionDecisionParams{ID: requestID, Status: next, ModeratorID: moderator}); err != nil {
			return err
		}
		if next == sqlc.DeletionStatusApproved {
			if err := q.ApproveImageDeletion(ctx, sqlc.ApproveImageDeletionParams{ImageID: imageID, RequestID: requestID}); err != nil {
				return err
			}
		} else if action == "REVOKE" {
			if err := q.RevokeImageDeletion(ctx, imageID); err != nil {
				return err
			}
		}
		if next == sqlc.DeletionStatusApproved || action == "REVOKE" {
			event := "delete"
			if action == "REVOKE" {
				event = "restore"
			}
			snapshot, err := q.GetImageSnapshot(ctx, imageID)
			if err != nil {
				return err
			}
			actor, err := q.GetOther(ctx, actorID)
			if err != nil {
				return err
			}
			payload, err := json.Marshal(Event{
				Event: event,
				Data: webhook.Data{ImageID: imageID, Text: snapshot.Text, Rating: sqlc.Rating(snapshot.Rating), Tags: snapshot.Tags,
					ImageURL: snapshot.ImageUrl, CreatedAt: snapshot.CreatedAt.Time, Uploader: &webhook.User{ID: snapshot.UserID, Username: snapshot.Username}},
				Initiator: &webhook.User{ID: actorID, Username: actor.Username}, Version: snapshot.Version,
			})
			if err != nil {
				return err
			}
			if err := q.InsertDeletionOutbox(ctx, sqlc.InsertDeletionOutboxParams{ImageID: imageID, Kind: "delete", Payload: payload}); err != nil {
				return err
			}
			return q.InsertDeletionOutbox(ctx, sqlc.InsertDeletionOutboxParams{ImageID: imageID, Kind: "search", Payload: []byte(`{}`)})
		}
		return nil
	})
	return normalizeError(err)
}

func normalizeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}
