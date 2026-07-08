package workers

type BulkUploadInput struct {
	BulkID       string
	ID           string
	FileKey      string
	BusinessID   string
	AggregatorID *string
	ServiceID    string
	IsSandbox    bool
}

const (
	BulkUploadTask = "bulk:upload"
)

const (
	BulkUploadStatusPending    = "pending"
	BulkUploadStatusProcessing = "processing"
	BulkUploadStatusCompleted  = "completed"
	BulkUploadStatusFailed     = "failed"
)
