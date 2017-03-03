package tasks

import (
	"strconv"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	tid "github.com/Financial-Times/transactionid-utils-go"
)

type Task interface {
	Publish(uuid string)
}

type nativeContentTask struct {
	nativeReader native.Reader
	cmsNotifier  cms.Notifier
}

const publishReferenceAttr = "publishReference"
const nativeHashHeader = "X-Native-Hash"

func (t *nativeContentTask) Publish(uuid string) {
	content, hash, err := t.nativeReader.Get(uuid)
	if err != nil {
		return
	}

	tid, ok := content.Body[publishReferenceAttr].(string)
	if !ok || strings.TrimSpace(tid) == "" {
		content.Body[publishReferenceAttr] = generateCarouselTXID()
	} else {
		content.Body[publishReferenceAttr] = toCarouselTXID(tid)
	}

	err = t.cmsNotifier.Notify(*content, hash)
	if err != nil {
		return
	}
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
