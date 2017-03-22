package tasks

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var carouselTidRegex = regexp.MustCompile(`^tid_[\S]+_carousel_[\d]{10}$`)
var carouselGentxTidRegex = regexp.MustCompile(`^tid_[\S]+_carousel_[\d]{10}_gentx$`)

var carouselTidMatcher = mock.MatchedBy(func(tid string) bool {
	strings.HasPrefix(tid, "tid_1234")
	return carouselTidRegex.Match([]byte(tid))
})

var carouselGentxTidMatcher = mock.MatchedBy(func(tid string) bool {
	strings.HasPrefix(tid, "tid_1234")
	return carouselGentxTidRegex.Match([]byte(tid))
})

func TestPublishWithTID(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	body := make(map[string]interface{})
	body[publishReferenceAttr] = "tid_1234"

	content := &native.Content{
		Body:        body,
		ContentType: "application/json",
	}

	reader.On("Get", testCollection, testUUID).Return(content, "hash", nil)
	notifier.On("Notify", origin, carouselTidMatcher, *content, "hash").Return(nil)

	task := NewNativeContentPublishTask(reader, notifier)

	err := task.Publish(origin, testCollection, testUUID)
	assert.NoError(t, err)
	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestPublishWithGeneratedTID(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	body := make(map[string]interface{})

	content := &native.Content{
		Body:        body,
		ContentType: "application/json",
	}

	reader.On("Get", testCollection, testUUID).Return(content, "hash", nil)
	notifier.On("Notify", origin, carouselGentxTidMatcher, *content, "hash").Return(nil)

	task := NewNativeContentPublishTask(reader, notifier)

	err := task.Publish(origin, testCollection, testUUID)
	assert.NoError(t, err)
	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestFailedReader(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	content := &native.Content{}

	reader.On("Get", testCollection, testUUID).Return(content, "hash", errors.New("fail"))

	task := NewNativeContentPublishTask(reader, notifier)

	err := task.Publish(origin, testCollection, testUUID)
	assert.Error(t, err)
	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestEmptyContent(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	content := &native.Content{}

	reader.On("Get", testCollection, testUUID).Return(content, "hash", nil)

	task := NewNativeContentPublishTask(reader, notifier)
	err := task.Publish(origin, testCollection, testUUID)
	assert.Error(t, err)
	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestFailedNotify(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	body := make(map[string]interface{})
	body[publishReferenceAttr] = "tid_1234"

	content := &native.Content{
		Body:        body,
		ContentType: "application/json",
	}

	reader.On("Get", testCollection, testUUID).Return(content, "hash", nil)
	notifier.On("Notify", origin, carouselTidMatcher, *content, "hash").Return(errors.New("fail"))

	task := NewNativeContentPublishTask(reader, notifier)

	err := task.Publish(origin, testCollection, testUUID)
	assert.Error(t, err)
	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
}
