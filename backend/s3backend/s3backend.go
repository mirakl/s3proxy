// S3 implementation for Backend Interface

package s3backend

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/mirakl/s3proxy/backend"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Config for s3backend for defining Host, Region, etc ...
type Config struct {
	Host             string
	Region           string
	AccessKey        string
	SecretKey        string
	DisableSSL       bool
	S3ForcePathStyle bool
}

type impl struct {
	client *s3.S3
	config Config
}

// New creates a new backend for s3 compatible backend
func New(config ...Config) (backend.Backend, error) {
	var s3Config *aws.Config
	var s3BackendConfig Config

	if len(config) > 1 {
		return nil, errors.New("One config max. allowed")
	} else if len(config) == 1 {
		s3BackendConfig = config[0]
		s3Config = &aws.Config{
			Credentials:      credentials.NewStaticCredentials(config[0].AccessKey, config[0].SecretKey, ""),
			Endpoint:         aws.String(config[0].Host),
			DisableSSL:       aws.Bool(config[0].DisableSSL),
			S3ForcePathStyle: aws.Bool(config[0].S3ForcePathStyle),
		}

		if config[0].Region != "" {
			s3Config.Region = aws.String(config[0].Region)
		}
	}

	var sess *session.Session
	var err error

	// Initialize a session
	if s3Config != nil {
		sess, err = session.NewSession(s3Config)
	} else {
		sess, err = session.NewSession()
	}

	if err != nil {
		return nil, err
	}

	// Create S3 service client
	return &impl{
		client: s3.New(sess),
		config: s3BackendConfig,
	}, nil
}

// CreatePresignedURLForUpload creates a presigned url for an upload of an object
func (b *impl) CreatePresignedURLForUpload(object backend.BucketObject, expire time.Duration) (string, error) {
	req, _ := b.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(object.BucketName),
		Key:    aws.String(object.Key),
	})

	res, err := req.Presign(expire)
	return res, mapErr(err)
}

// CreatePresignedURLForDownload creates a presigned url for a download of an object
func (b *impl) CreatePresignedURLForDownload(object backend.BucketObject, expire time.Duration) (string, error) {
	req, _ := b.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(object.BucketName),
		Key:    aws.String(object.Key),
	})

	res, err := req.Presign(expire)
	return res, mapErr(err)
}

// DeleteObject deletes an object in a bucket
func (b *impl) DeleteObject(object backend.BucketObject) error {

	_, err := b.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(object.BucketName),
		Key:    aws.String(object.Key),
	})

	return mapErr(err)
}

// BatchDeleteObjects deletes a list of objects in batch mode
func (b *impl) BatchDeleteObjects(objects []backend.BucketObject) error {
	batcher := s3manager.NewBatchDeleteWithClient(b.client)

	objectsToDelete := make([]s3manager.BatchDeleteObject, len(objects))

	for index, element := range objects {
		objectsToDelete[index] = s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Key:    aws.String(element.Key),
				Bucket: aws.String(element.BucketName),
			},
		}
	}

	err := batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
		Objects: objectsToDelete,
	})

	return mapErr(err)
}

// Copy item from source to destination bucket
func (b *impl) CopyObject(sourceObject backend.BucketObject, destinationObject backend.BucketObject) error {

	_, err := b.client.CopyObject(&s3.CopyObjectInput{
		CopySource: aws.String(sourceObject.FullPath()),
		Bucket:     aws.String(destinationObject.BucketName),
		Key:        aws.String(destinationObject.Key),
	})

	return mapErr(err)
}

func mapErr(err error) error {
	var e awserr.Error
	if ok := errors.As(err, &e); ok {
		switch e.Code() {
		case s3.ErrCodeNoSuchBucket:
			return backend.ErrBucketNotFound
		case s3.ErrCodeNoSuchKey:
			return backend.ErrObjectNotFound
		}
	}

	return err
}
