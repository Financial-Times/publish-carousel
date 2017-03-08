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

type S3ReadWrite interface {
	Write(id string, b []byte, contentType string) error
	Read(id string) (bool, io.ReadCloser, *string, error)
	GetLatestID() (string, error)
}

type DefaultS3RW struct {
	bucketName string
	s3api      s3iface.S3API
}

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

func (s *DefaultS3RW) Write(id string, b []byte, contentType string) error {
	params := &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(id),
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

func (s *DefaultS3RW) GetLatestID() (string, error) {
	output, err := s.s3api.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.bucketName),
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

func (s *DefaultS3RW) Read(id string) (bool, io.ReadCloser, *string, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(id),
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
