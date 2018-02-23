// Package backendtest provides utilities for backend testing
package backendtest

import (
	"s3proxy/backend"
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
func (b *S3FakeBackend) CreatePresignedURLForUpload(bucket string, key string, expire time.Duration) (string, error) {
	return b.s3Backend.CreatePresignedURLForUpload(bucket, key, expire)
}

// Create presigned url for download just like for a real s3 backend
func (b *S3FakeBackend) CreatePresignedURLForDownload(bucket string, key string, expire time.Duration) (string, error) {
	return b.s3Backend.CreatePresignedURLForDownload(bucket, key, expire)
}

// Delete action for an object in a bucket, in our case does nothing because no real backend
func (b *S3FakeBackend) DeleteObject(bucket string, key string) error {
	if strings.Contains(key, "error") {
		panic("Fake panic :))")
	}
	return nil
}
