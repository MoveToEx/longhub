package admin

import (
	"context"
	"long/internal/deletion"
	"long/internal/sqlc"
)

// Store keeps resolvers independent of the process-wide database connection.
type Store interface {
	AdminListImages(context.Context, sqlc.AdminListImagesParams) ([]sqlc.AdminListImagesRow, error)
	AdminCountImages(context.Context, sqlc.AdminCountImagesParams) (int64, error)
	AdminListUsers(context.Context, sqlc.AdminListUsersParams) ([]sqlc.AdminListUsersRow, error)
	AdminCountUsers(context.Context, sqlc.AdminCountUsersParams) (int64, error)
}

type DeletionStore interface {
	AdminListDeletionRequests(context.Context, sqlc.AdminListDeletionRequestsParams) ([]sqlc.DeletionRequestDetail, error)
	AdminCountDeletionRequests(context.Context, sqlc.AdminCountDeletionRequestsParams) (int64, error)
	GetDeletionRequestRecord(context.Context, int64) (sqlc.DeletionRequestDetail, error)
}

type Resolver struct {
	Store     Store
	Deletions *deletion.Service
}

//go:generate go tool gqlgen generate
