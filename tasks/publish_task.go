package tasks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/image"
	"github.com/Financial-Times/publish-carousel/native"
	tid "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/Sirupsen/logrus"
)

type Task interface {
	Prepare(collection string, uuid string) (*native.Content, string, error)
	Execute(uuid string, content *native.Content, origin string, txId string) error
}

type nativeContentTask struct {
	nativeReader native.Reader
	cmsNotifier  cms.Notifier
	isImage      image.Filter
}

// NewNativeContentPublishTask publishes the native content from mongo to the cms notifier, if the uuid has not been blacklisted.
func NewNativeContentPublishTask(reader native.Reader, notifier cms.Notifier, isImage image.Filter) Task {
	return &nativeContentTask{nativeReader: reader, cmsNotifier: notifier, isImage: isImage}
}

const publishReferenceAttr = "publishReference"

func (t *nativeContentTask) Prepare(collection string, uuid string) (*native.Content, string, error) {
	content, err := t.nativeReader.Get(collection, uuid)
	if err != nil {
		log.WithField("uuid", uuid).WithError(err).Warn("Failed to read from native reader")
		return nil, "", err
	}

	if content.Body == nil {
		log.WithField("uuid", uuid).Warn("No Content found for uuid. Skipping.")
		return nil, "", fmt.Errorf(`Skipping uuid "%v" as it has no content`, uuid)
	}

	invalid, err := t.isImage(uuid, content)
	if err != nil {
		log.WithField("uuid", uuid).WithField("collection", collection).WithError(err).Warn("Blacklist check failed.")
		return nil, "", err
	}

	if invalid {
		log.WithField("uuid", uuid).WithField("collection", collection).Info("This UUID contains an image. Skipping republish.")
		return nil, "", fmt.Errorf(`Skipping uuid "%v" as it is an image`, uuid)
	}

	tid, ok := content.Body[publishReferenceAttr].(string)
	if !ok || strings.TrimSpace(tid) == "" {
		tid = generateCarouselTXID()
	} else {
		tid = toCarouselTXID(tid)
	}

	return content, tid, nil
}

func (t *nativeContentTask) Execute(uuid string, content *native.Content, origin string, tid string) error {
	data, err := json.Marshal(content.Body)
	if err != nil {
		return err
	}

	hash, err := native.Hash(data)
	if err != nil {
		return err
	}

	content.Body[publishReferenceAttr] = tid

	err = t.cmsNotifier.Notify(origin, tid, content, hash)
	if err != nil {
		log.WithField("uuid", uuid).WithError(err).Warn("Failed to post to cms notifier")
		return err
	}

	return nil
}

const genTXSuffix = "_gentx"

func generateCarouselTXID() string {
	return tid.NewTransactionID() + generateTXIDSuffix() + genTXSuffix
}

func toCarouselTXID(tid string) string {
	return tid + generateTXIDSuffix()
}

const carouselIntrafix = "_carousel_"

func generateTXIDSuffix() string {
	return carouselIntrafix + strconv.FormatInt(time.Now().Unix(), 10)
}
