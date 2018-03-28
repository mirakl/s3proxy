// S3 implementation for Backend Interface

package backend

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"time"
)

// Config for s3backend for defining Host, Region, etc ...
type S3BackendConfig struct {
	Host             string
	Region           string
	AccessKey        string
	SecretKey        string
	DisableSSL       bool
	S3ForcePathStyle bool
}

// s3backend which will implement Backend interface
type S3Backend struct {
	client *s3.S3
	config S3BackendConfig
}

// Create a new backend for s3 compatible backend
func NewS3Backend(config ...S3BackendConfig) (Backend, error) {
	return newS3Backend(config)
}

func newS3Backend(config []S3BackendConfig) (*S3Backend, error) {

	var s3Config *aws.Config
	var s3BackendConfig S3BackendConfig

	if len((config)) > 1 {
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
	return &S3Backend{
		client: s3.New(sess),
		config: s3BackendConfig,
	}, nil
}

// Create a presigned url for an upload of an object
func (b *S3Backend) CreatePresignedURLForUpload(bucket string, key string, expire time.Duration) (string, error) {
	req, _ := b.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return req.Presign(expire)
}

// Create a presigned url for a download of an object
func (b *S3Backend) CreatePresignedURLForDownload(bucket string, key string, expire time.Duration) (string, error) {
	req, _ := b.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return req.Presign(expire)
}

// Delete action for an object in a bucket
func (b *S3Backend) DeleteObject(bucket string, key string) error {

	_, err := b.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}
	/* No synchronous wait for now
	err = b.client.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	*/
	return nil
}
