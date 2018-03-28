package backend

import "time"

// Interface for s3 backend
type Backend interface {
	// Create presigned URL for uploading file to the bucket
	CreatePresignedURLForUpload(bucket string, key string, expire time.Duration) (string, error)

	// Create presigned URL for downloading file from the bucket
	CreatePresignedURLForDownload(bucket string, key string, expire time.Duration) (string, error)

	// Delete object in a bucket
	DeleteObject(bucket string, key string) error
}
