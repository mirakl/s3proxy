package backend

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"
)

// Backend provides an interface for S3
type Backend interface {
	// CreatePresignedURLForUpload creates a presigned URL for uploading file to the bucket
	CreatePresignedURLForUpload(object BucketObject, expire time.Duration) (string, error)

	// CreatePresignedURLForDownload creates a presigned URL for downloading file from the bucket
	CreatePresignedURLForDownload(object BucketObject, expire time.Duration) (string, error)

	// DeleteObject delete an object in a bucket
	DeleteObject(object BucketObject) error

	// BatchDeleteObject delete an object in a bucket
	BatchDeleteObjects(objects []BucketObject) error

	// CopyObject copies one item from a bucket to another
	// sourceObject : source object (ex: mybucket and /folder/item)
	// destinationObject : destination object (ex: mybucket and /folder/item2)
	CopyObject(sourceObject BucketObject, destinationObject BucketObject) error
}

// BucketObject is a tuple containing an object key (ex: /folder/item) and a bucket name (ex: mybucket)
type BucketObject struct {
	BucketName string
	Key        string
}

func (b BucketObject) String() string {
	return fmt.Sprintf("%s (%s)", b.BucketName, b.Key)
}

func (b BucketObject) FullPath() string {
	return fmt.Sprintf("/%s%s", b.BucketName, b.Key)
}

var (
	ErrBucketNotFound = errors.New("No such bucket")
	ErrObjectNotFound = errors.New("No such key")
)
