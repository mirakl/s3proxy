// Package backendtest provides utilities for backend testing
package backendtest

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mirakl/s3proxy/backend"
	"strings"
	"time"
)

// S3 implementation for a fake Backend Interface, all methods are the same except delete
func NewS3FakeBackend(config ...backend.S3BackendConfig) (backend.Backend, error) {
	return newS3FakeBackend(config)
}

func newS3FakeBackend(config []backend.S3BackendConfig) (*S3FakeBackend, error) {

	var s3backend backend.Backend
	var err error

	if config == nil || len(config) == 0 {
		s3backend, err = backend.NewS3Backend()
	} else {
		s3backend, err = backend.NewS3Backend(config[0])
	}

	if err != nil {
		return nil, err
	}

	// Create S3 service client
	return &S3FakeBackend{
		s3Backend: s3backend,
	}, nil
}

// Fake s3 backend implementation
type S3FakeBackend struct {
	s3Backend backend.Backend
}

// Create presigned url for upload just like for a real s3 backend
func (b *S3FakeBackend) CreatePresignedURLForUpload(object backend.BucketObject, expire time.Duration) (string, error) {
	return b.s3Backend.CreatePresignedURLForUpload(object, expire)
}

// Create presigned url for download just like for a real s3 backend
func (b *S3FakeBackend) CreatePresignedURLForDownload(object backend.BucketObject, expire time.Duration) (string, error) {
	return b.s3Backend.CreatePresignedURLForDownload(object, expire)
}

// Delete action for an object in a bucket, in our case does nothing because no real backend
func (b *S3FakeBackend) DeleteObject(object backend.BucketObject) error {
	if strings.Contains(object.Key, "error") {
		panic("Fake panic :))")
	}
	return nil
}

func (b *S3FakeBackend) BatchDeleteObjects(objects []backend.BucketObject) error {
	return nil
}

// Fake copy, does nothing except returning some errors when the keyword "notfound" are used
func (b *S3FakeBackend) CopyObject(sourceObject backend.BucketObject, destinationObject backend.BucketObject) error {
	if strings.Contains(sourceObject.BucketName, "notfound") || strings.Contains(destinationObject.BucketName, "notfound") {
		return awserr.New(s3.ErrCodeNoSuchBucket, "No such bucket", nil)
	}
	if strings.Contains(sourceObject.Key, "notfound") {
		return awserr.New(s3.ErrCodeNoSuchKey, "No such key", nil)
	}
	return nil
}
