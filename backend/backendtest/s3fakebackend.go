// Package backendtest provides utilities for backend testing
package backendtest

import (
	"strings"
	"time"

	"github.com/mirakl/s3proxy/backend/s3backend"

	"github.com/mirakl/s3proxy/backend"
)

// New S3 implementation for a fake Backend Interface, all methods are the same except delete
func New(config ...s3backend.Config) (backend.Backend, error) {
	var s3Backend backend.Backend
	var err error

	if len(config) == 0 {
		s3Backend, err = s3backend.New()
	} else {
		s3Backend, err = s3backend.New(config[0])
	}

	if err != nil {
		return nil, err
	}

	// Create S3 service client
	return &impl{
		s3Backend: s3Backend,
	}, nil
}

// Fake s3 backend implementation
type impl struct {
	s3Backend backend.Backend
}

// CreatePresignedURLForUpload creates presigned url for upload just like for a real s3 backend
func (b *impl) CreatePresignedURLForUpload(object backend.BucketObject, expire time.Duration) (string, error) {
	return b.s3Backend.CreatePresignedURLForUpload(object, expire)
}

// CreatePresignedURLForDownload creates presigned url for download just like for a real s3 backend
func (b *impl) CreatePresignedURLForDownload(object backend.BucketObject, expire time.Duration) (string, error) {
	return b.s3Backend.CreatePresignedURLForDownload(object, expire)
}

// DeleteObject delete action for an object in a bucket, in our case does nothing because no real backend
func (b *impl) DeleteObject(object backend.BucketObject) error {
	if strings.Contains(object.Key, "error") {
		panic("Fake panic :))")
	}
	return nil
}

func (b *impl) BatchDeleteObjects(objects []backend.BucketObject) error {
	return nil
}

// Fake copy, does nothing except returning some errors when the keyword "notfound" are used
func (b *impl) CopyObject(sourceObject backend.BucketObject, destinationObject backend.BucketObject) error {
	if strings.Contains(sourceObject.BucketName, "notfound") || strings.Contains(destinationObject.BucketName, "notfound") {
		return backend.ErrBucketNotFound
	}
	if strings.Contains(sourceObject.Key, "notfound") {
		return backend.ErrObjectNotFound
	}
	return nil
}
