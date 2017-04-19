package tasks

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	tid "github.com/Financial-Times/transactionid-utils-go"
	log "github.com/Sirupsen/logrus"
)

type Task interface {
	Publish(origin string, collection string, uuid string) error
}

type nativeContentTask struct {
	nativeReader native.Reader
	cmsNotifier  cms.Notifier
	blacklist    blacklist.Blacklist
}

// NewNativeContentPublishTask publishes the native content from mongo to the cms notifier, if the uuid has not been blacklisted.
func NewNativeContentPublishTask(reader native.Reader, notifier cms.Notifier, blist blacklist.Blacklist) Task {
	return &nativeContentTask{nativeReader: reader, cmsNotifier: notifier, blacklist: blist}
}

const publishReferenceAttr = "publishReference"

func (t *nativeContentTask) Publish(origin string, collection string, uuid string) error {
	content, hash, err := t.nativeReader.Get(collection, uuid)
	if err != nil {
		log.WithField("uuid", uuid).WithError(err).Warn("Failed to read from native reader")
		return err
	}

	if content.Body == nil {
		log.WithField("uuid", uuid).Warn("No Content found for uuid. Skipping.")
		return fmt.Errorf(`Skipping uuid "%v" as it has no content`, uuid)
	}

	valid, err := t.blacklist.ValidForPublish(uuid, content)
	if err != nil {
		log.WithField("uuid", uuid).WithField("collection", collection).WithError(err).Warn("Blacklist check failed.")
		return err
	}

	if !valid {
		log.WithField("uuid", uuid).WithField("collection", collection).Info("This UUID has been blacklisted. Skipping republish.")
		return nil
	}

	tid, ok := content.Body[publishReferenceAttr].(string)
	if !ok || strings.TrimSpace(tid) == "" {
		tid = generateCarouselTXID()
	} else {
		tid = toCarouselTXID(tid)
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
