package admin

import (
	"context"
	"errors"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"log"
	"long/internal/admin/model"
	"long/internal/deletion"
	"long/internal/queue"
	"long/internal/sqlc"
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
	store, ok := r.Store.(DeletionStore)
	if !ok {
		return nil, gqlerror.Errorf("Unable to load deletion requests")
	}
	search, pattern, id := searchParams(input.Search)
	rows, err := store.AdminListDeletionRequests(ctx, sqlc.AdminListDeletionRequestsParams{Status: input.Status.String(), Search: search, Pattern: pattern, SearchID: id,
		SortBy: input.SortBy.String(), Direction: input.Direction.String(), PageLimit: int32(input.Limit), PageOffset: int32(input.Offset)})
	if err != nil {
		return nil, queryError(ctx, err)
	}
	total, err := store.AdminCountDeletionRequests(ctx, sqlc.AdminCountDeletionRequestsParams{Status: input.Status.String(), Search: search, Pattern: pattern, SearchID: id})
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
	store, ok := r.Store.(DeletionStore)
	if !ok || r.Deletions == nil {
		return nil, gqlerror.Errorf("Unable to moderate deletion request")
	}
	err = r.Deletions.Moderate(ctx, requestID, actor, action.String(), sqlc.DeletionStatus(strings.ToLower(expected.String())))
	if err != nil {
		if errors.Is(err, deletion.ErrConflict) {
			return nil, &gqlerror.Error{Message: err.Error(), Extensions: map[string]any{"code": "CONFLICT"}}
		}
		if errors.Is(err, deletion.ErrNotFound) {
			return nil, &gqlerror.Error{Message: err.Error(), Extensions: map[string]any{"code": "NOT_FOUND"}}
		}
		return nil, queryError(ctx, err)
	}
	if err := queue.WakeDeletionOutbox(ctx); err != nil {
		log.Printf("wake deletion outbox (scheduled retry remains active): %v", err)
	}
	row, err := store.GetDeletionRequestRecord(ctx, requestID)
	if err != nil {
		return nil, queryError(ctx, err)
	}
	return deletionModel(row), nil
}
