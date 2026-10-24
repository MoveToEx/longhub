package queue

// Expose queue internals to integration tests in queue_test so they can use the
// admin package without introducing an import cycle.
var (
	SyncImageIndexForTest         = syncImageIndex
	DispatchDeletionOutboxForTest = dispatchDeletionOutbox
)
