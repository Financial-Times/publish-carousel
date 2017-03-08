package s3

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

const fileFormat = ""

// S3ReadWrite is responsible for reading, writing and locating the latest cycle restore files from S3
type S3ReadWrite interface {
	Write(id string, b []byte, contentType string) error
	Read(key string) (bool, io.ReadCloser, *string, error)
	GetLatestKeyForID(id string) (string, error)
}

// DefaultS3RW the default S3ReadWrite implementation
type DefaultS3RW struct {
	bucketName string
	s3api      s3iface.S3API
}

// NewS3ReadWrite create a new S3 R/W for the given region and bucket
func NewS3ReadWrite(region string, bucketName string) (S3ReadWrite, error) {
	hc := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          20,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   20,
			TLSHandshakeTimeout:   3 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	sess, err := session.NewSession(
		&aws.Config{
			Region:     aws.String(region),
			MaxRetries: aws.Int(1),
			HTTPClient: &hc,
		})

	if err != nil {
		return nil, err
	}

	s3api := s3.New(sess)
	return &DefaultS3RW{bucketName: bucketName, s3api: s3api}, nil
}

// Write writes the given ID to S3
func (s *DefaultS3RW) Write(id string, b []byte, contentType string) error {
	timestamp := time.Now().UTC().Format(time.UnixDate)
	params := &s3.PutObjectInput{
		Bucket: aws.String(s.getBucketForID(id)),
		Key:    aws.String(timestamp),
		Body:   bytes.NewReader(b),
	}

	if strings.TrimSpace(contentType) != "" {
		params.ContentType = aws.String(contentType)
	}

	resp, err := s.s3api.PutObject(params)

	if err != nil {
		log.Infof("Error found, Resp was : %v", resp)
		return err
	}

	return nil
}

func (s *DefaultS3RW) GetLatestKeyForID(id string) (string, error) {
	output, err := s.s3api.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.getBucketForID(id)),
	})

	if err != nil {
		return "", err
	}

	var latestKey string
	var latestTimestamp *time.Time
	for _, obj := range output.Contents {
		if latestTimestamp == nil || latestTimestamp.Before(*obj.LastModified) {
			latestTimestamp = obj.LastModified
			latestKey = *obj.Key
		}
	}

	return latestKey, nil
}

func (s *DefaultS3RW) getBucketForID(id string) string {
	return s.bucketName + "/" + id + "/"
}

func (s *DefaultS3RW) Read(key string) (bool, io.ReadCloser, *string, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	resp, err := s.s3api.GetObject(params)

	if err != nil {
		e, ok := err.(awserr.Error)
		if ok && e.Code() == "NoSuchKey" {
			return false, nil, nil, nil
		}
		return false, nil, nil, err
	}

	return true, resp.Body, resp.ContentType, err
}
