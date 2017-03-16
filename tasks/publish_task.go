package tasks

import (
	"strconv"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	tid "github.com/Financial-Times/transactionid-utils-go"
	"github.com/Sirupsen/logrus"
)

type Task interface {
	Publish(origin string, collection string, uuid string) error
}

type nativeContentTask struct {
	nativeReader native.Reader
	cmsNotifier  cms.Notifier
}

func NewNativeContentPublishTask(reader native.Reader, notifier cms.Notifier) Task {
	return &nativeContentTask{reader, notifier}
}

const publishReferenceAttr = "publishReference"
const nativeHashHeader = "X-Native-Hash"

func (t *nativeContentTask) Publish(origin string, collection string, uuid string) error {
	content, hash, err := t.nativeReader.Get(collection, uuid)
	if err != nil {
		logrus.WithField("uuid", uuid).WithError(err).Error("Failed to read from native reader")
		return err
	}

	tid, ok := content.Body[publishReferenceAttr].(string)
	if !ok || strings.TrimSpace(tid) == "" {
		content.Body[publishReferenceAttr] = generateCarouselTXID()
	} else {
		content.Body[publishReferenceAttr] = toCarouselTXID(tid)
	}

	err = t.cmsNotifier.Notify(origin, tid, *content, hash)
	if err != nil {
		logrus.WithField("uuid", uuid).WithError(err).Error("Failed to post to cms notifier")
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
