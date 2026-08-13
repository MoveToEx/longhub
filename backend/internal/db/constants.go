package db

const (
	WebhookCreationEvent = 1 << iota
	WebhookUpdateEvent
	WebhookDeletionEvent
)

const (
	PermissionCreate int64 = 1 << iota
	PermissionEdit
	PermissionRequestDeletion
	PermissionAdmin
	PermissionApproveDeletion
	PermissionRenameTag
	PermissionGrant
	PermissionManageUser
)
