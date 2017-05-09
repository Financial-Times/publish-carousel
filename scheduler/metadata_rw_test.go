package scheduler

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/Financial-Times/publish-carousel/s3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWriteMetadata(t *testing.T) {

	cfg := CycleConfig{
		Name:       "test-cycle",
		Type:       "test-type",
		Origin:     "test-origin",
		Collection: "test-collection",
		CoolDown:   "test-cooldown",
		Throttle:   "test-throttle",
	}

	md := CycleMetadata{
		CurrentPublishUUID: "00000000-0000-0000-0000-000000000000",
		Errors:             1,
		Progress:           0.5,
		State:              []string{},
		Completed:          3,
		Total:              4,
		Iteration:          5,
	}

	id := "test-cycle-id"
	cycle := new(MockCycle)
	cycle.On("TransformToConfig").Return(&cfg)
	cycle.On("Metadata").Return(md)

	s3rw := new(s3.MockReadWriter)
	s3rw.On("Write",
		id,
		mock.MatchedBy(func(actual string) bool { return regexp.MustCompile(`\d{8}T\d{8}`).MatchString(actual) }),
		mock.MatchedBy(func(actual []byte) bool { return true }),
		"application/json").Return(nil)

	rw := s3MetadataReadWriter{s3rw}
	err := rw.WriteMetadata(id, cycle)

	assert.NoError(t, err)
	s3rw.AssertExpectations(t)
}

func TestRead(t *testing.T) {
	id := "test-cycle-id"
	key := "test-key"
	contentType := "application/json"

	uuid := "00000000-0000-0000-0000-000000000000"
	errors := 1
	progress := 0.5
	completed := 2
	total := 3
	iteration := 4
	state := fmt.Sprintf(`{
		"config": {
			"name":"test-cycle",
			"type":"test-type",
			"origin":"test-origin",
			"collection":"test-collection",
			"coolDown":"1m0s"
		},
		"metadata": {
			"currentPublishUuid":"%s",
			"errors":%d,
			"progress":%f,
			"state":[],
			"completed":%d,
			"total":%d,
			"iteration":%d
		}
	}`, uuid, errors, progress, completed, total, iteration)

	s3rw := new(s3.MockReadWriter)
	s3rw.On("GetLatestKeyForID", id).Return(key, nil)
	s3rw.On("Read", key).Return(true, nopCloser{strings.NewReader(state)}, &contentType, nil)

	rw := s3MetadataReadWriter{s3rw}
	md, err := rw.LoadMetadata(id)

	assert.NoError(t, err)

	assert.Equal(t, uuid, md.CurrentPublishUUID, "current publish UUID")
	assert.Equal(t, errors, md.Errors, "errors")
	assert.Equal(t, progress, md.Progress, "progress")
	assert.Equal(t, completed, md.Completed, "completed")
	assert.Equal(t, total, md.Total, "total")
	assert.Equal(t, iteration, md.Iteration, "iteration")

	s3rw.AssertExpectations(t)
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
