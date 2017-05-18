package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/s3"
)

const defaultContentType = "application/json"

type MetadataReadWriter interface {
	LoadMetadata(id string) (CycleMetadata, error)
	WriteMetadata(id string, config CycleConfig, metadata CycleMetadata) error
}

type s3MetadataReadWriter struct {
	s3rw s3.ReadWriter
}

type s3Metadata struct {
	Config   CycleConfig   `json:"config"`
	Metadata CycleMetadata `json:"metadata"`
}

func NewS3MetadataReadWriter(rw s3.ReadWriter) MetadataReadWriter {
	return &s3MetadataReadWriter{s3rw: rw}
}

func (s *s3MetadataReadWriter) LoadMetadata(id string) (CycleMetadata, error) {
	key, err := s.s3rw.GetLatestKeyForID(id)
	if err != nil {
		return CycleMetadata{}, err
	}

	if strings.TrimSpace(key) == "" {
		return CycleMetadata{}, errors.New(`No key found for id "` + id + `"`)
	}

	found, body, contentType, err := s.s3rw.Read(key)
	if err != nil {
		return CycleMetadata{}, err
	}

	if !found {
		return CycleMetadata{}, fmt.Errorf(`No state found for "%v"`, id)
	}

	if contentType == nil || strings.TrimSpace(*contentType) != "application/json" {
		return CycleMetadata{}, fmt.Errorf(`Failed to load state for "%v". Content was in an unexpected Content-Type "%v"`, id, contentType)
	}

	fromS3 := &s3Metadata{}
	dec := json.NewDecoder(body)
	err = dec.Decode(fromS3)

	return fromS3.Metadata, err
}

func (s *s3MetadataReadWriter) WriteMetadata(id string, config CycleConfig, metadata CycleMetadata) error {
	b, err := json.Marshal(&s3Metadata{config, metadata})
	if err != nil {
		return err
	}

	key := time.Now().UTC().Format(`20060102T15040599`)
	return s.s3rw.Write(id, key, b, defaultContentType)
}
