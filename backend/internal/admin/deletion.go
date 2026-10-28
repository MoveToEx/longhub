package admin

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"log"
	"long/internal/admin/model"
	"long/internal/db"
	"long/internal/queue"
	"long/internal/sqlc"
	"long/internal/webhook"
	"strconv"
	"strings"
)

func deletionModel(row sqlc.DeletionRequestDetail) *model.DeletionRequest {
	result := &model.DeletionRequest{ID: strconv.FormatInt(row.ID, 10), ImageID: strconv.FormatInt(row.ImageID, 10),
		Reason: row.Reason, Status: model.DeletionStatus(strings.ToUpper(string(row.Status))),
		Requester: &model.AdminUploader{ID: strconv.FormatInt(row.UserID, 10), Username: row.RequesterUsername}, CreatedAt: row.CreatedAt.Time}
	if row.ProcessedBy.Valid {
		result.Moderator = &model.AdminUploader{ID: strconv.FormatInt(row.ProcessedBy.Int64, 10), Username: row.ModeratorUsername.String}
	}
	if row.ProcessedAt.Valid {
		result.ProcessedAt = &row.ProcessedAt.Time
	}
	if row.ImageDeletedAt.Valid {
		result.ImageDeletedAt = &row.ImageDeletedAt.Time
	}
	if row.ActiveDeletionID.Valid {
		id := strconv.FormatInt(row.ActiveDeletionID.Int64, 10)
		result.ActiveDeletionID = &id
	}
	return result
}

func (r *Resolver) listDeletionRequests(ctx context.Context, input *model.DeletionQueryInput) (*model.DeletionRequestPage, error) {
	if input == nil {
		input = &model.DeletionQueryInput{Limit: 25, Status: model.DeletionStatusFilterActive, SortBy: model.DeletionSortCreatedAt, Direction: model.SortDirectionDesc}
	}
	if err := validatePage(input.Offset, input.Limit, input.Search); err != nil {
		return nil, err
	}
	search, pattern, id := searchParams(input.Search)
	rows, err := db.Query().AdminListDeletionRequests(ctx, sqlc.AdminListDeletionRequestsParams{Status: input.Status.String(), Search: search, Pattern: pattern, SearchID: id,
		SortBy: input.SortBy.String(), Direction: input.Direction.String(), PageLimit: int32(input.Limit), PageOffset: int32(input.Offset)})
	if err != nil {
		return nil, queryError(ctx, err)
	}
	total, err := db.Query().AdminCountDeletionRequests(ctx, sqlc.AdminCountDeletionRequestsParams{Status: input.Status.String(), Search: search, Pattern: pattern, SearchID: id})
	if err != nil {
		return nil, queryError(ctx, err)
	}
	result := &model.DeletionRequestPage{Items: make([]*model.DeletionRequest, 0, len(rows)), Total: total}
	for _, row := range rows {
		result.Items = append(result.Items, deletionModel(row))
	}
	return result, nil
}

func (r *Resolver) moderateDeletion(ctx context.Context, id string, action model.DeletionAction, expected model.DeletionStatus) (*model.DeletionRequest, error) {
	actor, _ := ctx.Value(actorContextKey{}).(int64)
	if actor <= 0 {
		return nil, &gqlerror.Error{Message: "Authentication required", Extensions: map[string]any{"code": "UNAUTHENTICATED"}}
	}
	requestID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || requestID <= 0 {
		return nil, gqlerror.Errorf("Invalid deletion request id")
	}
	conflict := errors.New("request or image state changed; refresh and try again")
	var deletionEvent *webhook.DeletionEvent
	err = db.Transaction(ctx, func(q *sqlc.Queries) error {
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
		if request.Status != sqlc.DeletionStatus(strings.ToLower(expected.String())) {
			return conflict
		}
		var next sqlc.DeletionStatus
		switch action {
		case model.DeletionActionAccept:
			if request.Status == sqlc.DeletionStatusActive || request.Status == sqlc.DeletionStatusRejected {
				next = sqlc.DeletionStatusApproved
			}
		case model.DeletionActionReject:
			if request.Status == sqlc.DeletionStatusActive {
				next = sqlc.DeletionStatusRejected
			}
		case model.DeletionActionReopen:
			if request.Status == sqlc.DeletionStatusRejected {
				next = sqlc.DeletionStatusActive
			}
		}
		if next == "" {
			return conflict
		}
		if next == sqlc.DeletionStatusApproved && (image.DeletedAt.Valid || image.ActiveDeletionID.Valid) {
			return conflict
		}
		moderator := pgtype.Int8{Int64: actor, Valid: next != sqlc.DeletionStatusActive}
		if err := q.UpdateDeletionDecision(ctx, sqlc.UpdateDeletionDecisionParams{ID: requestID, Status: next, ModeratorID: moderator}); err != nil {
			return err
		}
		if next == sqlc.DeletionStatusApproved {
			if err := q.ApproveImageDeletion(ctx, sqlc.ApproveImageDeletionParams{ImageID: imageID, RequestID: requestID}); err != nil {
				return err
			}
			snapshot, err := q.GetImageSnapshot(ctx, imageID)
			if err != nil {
				return err
			}
			initiator, err := q.GetOther(ctx, actor)
			if err != nil {
				return err
			}
			deletionEvent = &webhook.DeletionEvent{
				Data: webhook.Data{DeletionEventID: "deletion-request-" + strconv.FormatInt(requestID, 10),
					ImageID: imageID, Text: snapshot.Text, Rating: sqlc.Rating(snapshot.Rating), Tags: snapshot.Tags,
					ImageURL: snapshot.ImageUrl, CreatedAt: snapshot.CreatedAt.Time, Uploader: &webhook.User{ID: snapshot.UserID, Username: snapshot.Username}},
				Initiator: &webhook.User{ID: actor, Username: initiator.Username}, Version: snapshot.Version,
			}
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, conflict) || errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, &gqlerror.Error{Message: conflict.Error(), Extensions: map[string]any{"code": "CONFLICT"}}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &gqlerror.Error{Message: "image or deletion request not found", Extensions: map[string]any{"code": "NOT_FOUND"}}
		}
		return nil, queryError(ctx, err)
	}
	if deletionEvent != nil {
		if err := queue.EnqueueDeletion(ctx, *deletionEvent); err != nil {
			log.Printf("enqueue image deletion %d: %v", deletionEvent.Data.ImageID, err)
		}
	}
	row, err := db.Query().GetDeletionRequestRecord(ctx, requestID)
	if err != nil {
		return nil, queryError(ctx, err)
	}
	return deletionModel(row), nil
}
