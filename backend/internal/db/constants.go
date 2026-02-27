package db

const (
	WebhookCreationEvent = 1 << iota
	WebhookUpdateEvent
	WebhookDeletionEvent
)

const (
	PermissionEdit int64 = 1 << iota
	PermissionCreate
	PermissionRequestDeletion
	PermissionAdmin
	PermissionApproveDeletion
	PermissionRenameTag
	PermissionGrant
	PermissionManageUser
)
