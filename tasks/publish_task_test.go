package tasks

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func mockContent(publishRef string) (*native.Content, string) {
	body := make(map[string]interface{})
	if publishRef != "" {
		body[publishReferenceAttr] = publishRef
	}

	content := &native.Content{
		Body:        body,
		ContentType: "application/json",
	}

	data, _ := json.Marshal(body)
	hash, _ := native.Hash(data)

	return content, hash
}

func TestPublishWithTID(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	content, hash := mockContent("tid_1234")

	reader.On("Get", testCollection, testUUID).Return(content, nil)
	notifier.On("Notify", origin, carouselTidMatcher, content, hash).Return(nil)
	blist.On("ValidForPublish", testUUID, content).Return(true, nil)

	task := NewNativeContentPublishTask(reader, notifier, blist)

	content, txId, err := task.Prepare(testCollection, testUUID)
	require.NoError(t, err)

	err = task.Execute(testUUID, content, origin, txId)
	assert.NoError(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}

func TestPublishWithGeneratedTID(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	content, hash := mockContent("")

	reader.On("Get", testCollection, testUUID).Return(content, nil)
	notifier.On("Notify", origin, carouselGentxTidMatcher, content, hash).Return(nil)
	blist.On("ValidForPublish", testUUID, content).Return(true, nil)

	task := NewNativeContentPublishTask(reader, notifier, blist)

	content, txId, err := task.Prepare(testCollection, testUUID)
	require.NoError(t, err)

	err = task.Execute(testUUID, content, origin, txId)
	assert.NoError(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}

func TestPublishJSONMarshalFails(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testUUID := "fake-uuid"
	origin := "fake-origin"
	txId := "tid_1234"

	testBody := make(map[string]interface{})
	testBody["errrr"] = func() {}
	content := native.Content{Body: testBody, ContentType: "application/vnd.expect-this"}

	task := NewNativeContentPublishTask(reader, notifier, blist)

	err := task.Execute(testUUID, &content, origin, txId)
	assert.Error(t, err)
}

func TestFailedReader(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"

	content := &native.Content{}

	reader.On("Get", testCollection, testUUID).Return(content, errors.New("fail"))

	task := NewNativeContentPublishTask(reader, notifier, blist)

	_, _, err := task.Prepare(testCollection, testUUID)
	assert.Error(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}

func TestEmptyContent(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"

	content := &native.Content{}

	reader.On("Get", testCollection, testUUID).Return(content, nil)

	task := NewNativeContentPublishTask(reader, notifier, blist)
	_, _, err := task.Prepare(testCollection, testUUID)
	assert.Error(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}

func TestFailedNotify(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"
	origin := "fake-origin"

	content, hash := mockContent("tid_1234")

	reader.On("Get", testCollection, testUUID).Return(content, nil)
	notifier.On("Notify", origin, carouselTidMatcher, content, hash).Return(errors.New("fail"))
	blist.On("ValidForPublish", testUUID, content).Return(true, nil)

	task := NewNativeContentPublishTask(reader, notifier, blist)

	content, txId, err := task.Prepare(testCollection, testUUID)
	assert.NoError(t, err)

	err = task.Execute(testUUID, content, origin, txId)
	assert.Error(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}

func TestBlacklistedUUID(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"

	body := make(map[string]interface{})
	body[publishReferenceAttr] = "tid_1234"

	testContent := &native.Content{
		Body:        body,
		ContentType: "application/json",
	}

	reader.On("Get", testCollection, testUUID).Return(testContent, nil)
	blist.On("ValidForPublish", testUUID, testContent).Return(false, nil)

	task := NewNativeContentPublishTask(reader, notifier, blist)

	_, _, err := task.Prepare(testCollection, testUUID)
	require.Error(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}

func TestBlacklistFails(t *testing.T) {
	notifier := new(cms.MockNotifier)
	reader := new(native.MockReader)
	blist := new(blacklist.MockBlacklist)

	testCollection := "testing123"
	testUUID := "i am a uuid"

	body := make(map[string]interface{})
	body[publishReferenceAttr] = "tid_1234"

	content := &native.Content{
		Body:        body,
		ContentType: "application/json",
	}

	reader.On("Get", testCollection, testUUID).Return(content, nil)
	blist.On("ValidForPublish", testUUID, content).Return(false, errors.New("oh dear"))

	task := NewNativeContentPublishTask(reader, notifier, blist)

	_, _, err := task.Prepare(testCollection, testUUID)
	assert.Error(t, err)

	reader.AssertExpectations(t)
	notifier.AssertExpectations(t)
	blist.AssertExpectations(t)
}
