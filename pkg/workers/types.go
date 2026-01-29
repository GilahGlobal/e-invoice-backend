package workers

type BulkUploadInput struct {
	ID         string
	FileKey    string
	BusinessID string
	ServiceID  string
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
