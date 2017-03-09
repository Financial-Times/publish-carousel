package s3

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// ReadWriter is responsible for reading, writing and locating the latest cycle restore files from S3
type ReadWriter interface {
	Write(id string, key string, b []byte, contentType string) error
	Read(key string) (bool, io.ReadCloser, *string, error)
	GetLatestKeyForID(id string) (string, error)
	Ping() error
}

// DefaultReadWriter the default S3ReadWrite implementation
type DefaultReadWriter struct {
	bucketName string
	session    *session.Session
	config     *aws.Config
	lock       *sync.Mutex
}

// NewReadWriter create a new S3 R/W for the given region and bucket
func NewReadWriter(region string, bucketName string) ReadWriter {
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

	conf := &aws.Config{
		Region:     aws.String(region),
		MaxRetries: aws.Int(1),
		HTTPClient: &hc,
	}

	return &DefaultReadWriter{bucketName: bucketName, config: conf, lock: &sync.Mutex{}}
}

// Ping checks whether an S3 session has been initialised
func (s *DefaultReadWriter) Ping() error {
	if s.session == nil {
		return errors.New("S3 session is not initialised!")
	}
	return nil
}

func (s *DefaultReadWriter) open() (s3iface.S3API, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.session == nil {
		sess, err := session.NewSession(s.config)
		if err != nil {
			return nil, err
		}

		s.session = sess
	}

	return s3.New(s.session), nil
}

// Write writes the given ID to S3
func (s *DefaultReadWriter) Write(id string, key string, b []byte, contentType string) error {
	s3api, err := s.open()
	if err != nil {
		return err
	}

	params := &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(b),
	}

	if strings.TrimSpace(contentType) != "" {
		params.ContentType = aws.String(contentType)
	}

	resp, err := s3api.PutObject(params)

	if err != nil {
		log.WithField("response", resp).Info("Failed to write object to S3.", resp)
		return err
	}

	return nil
}

// GetLatestKeyForID lists s3 objects in the folder for the given ID
func (s *DefaultReadWriter) GetLatestKeyForID(id string) (string, error) {
	s3api, err := s.open()
	if err != nil {
		return "", err
	}

	output, err := s3api.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(s.getPrefixForID(id)),
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

func (s *DefaultReadWriter) getPrefixForID(id string) string {
	return id + "/"
}

// Read reads the provided key and returns a reader etc.
func (s *DefaultReadWriter) Read(key string) (bool, io.ReadCloser, *string, error) {
	s3api, err := s.open()
	if err != nil {
		return false, nil, nil, err
	}

	log.WithField("key", key).Info("Reading object from S3.")
	params := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	resp, err := s3api.GetObject(params)

	if err != nil {
		e, ok := err.(awserr.Error)
		if ok && e.Code() == "NoSuchKey" {
			return false, nil, nil, nil
		}
		return false, nil, nil, err
	}

	return true, resp.Body, resp.ContentType, err
}
