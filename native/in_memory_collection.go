package native

import (
	"context"
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/s3"
	log "github.com/sirupsen/logrus"
)

type InMemoryUUIDCollection struct {
	uuids      []string
	collection string
	skip       int
}

type InMemoryCollectionBuilder struct {
	s3ReadWriter s3.ReadWriter
}

func NewInMemoryCollectionBuilder(s3ReadWriter s3.ReadWriter) *InMemoryCollectionBuilder {
	return &InMemoryCollectionBuilder{s3ReadWriter: s3ReadWriter}
}

func (b *InMemoryCollectionBuilder) LoadIntoMemory(ctx context.Context, uuidCollection UUIDCollection, collection string, skip int, blist blacklist.IsBlacklisted) (UUIDCollection, error) {
	defer uuidCollection.Close()

	if skip > 0 && b.s3ReadWriter != nil {
		log.WithField("collection", collection).Info("Attempting to retrieve uuids from S3")
		uuids, err := readFromS3(b.s3ReadWriter, collection)
		if err != nil {
			log.WithError(err).WithField("collection", collection).Warn("Failed to retrieve persisted file from S3")
		} else if len(uuids) > 0 {
			if skip < len(uuids) {
				return &InMemoryUUIDCollection{collection: collection, skip: skip, uuids: uuids[skip:]}, nil
			}
			log.WithField("skip", skip).WithField("uuids", len(uuids)).Info("Unexpected value for skip! It's greater than the total number of uuids to process. Restarting from zero.")
			skip = 0
		}
	}

	it := &InMemoryUUIDCollection{collection: collection, skip: skip, uuids: make([]string, 0)}

	if uuidCollection.Length() == 0 {
		log.WithField("collection", collection).Warn("No data in mongo cursor for this collection.")
		return it, nil
	}

	log.WithField("collection", collection).WithField("skip", skip).Info("Loading collection into memory...")

	i := 0
	blank := 0
	blacklisted := 0

	overallStart := time.Now()
	start := time.Now()
	var end time.Time

	for {
		if ctx.Err() != nil {
			log.WithError(ctx.Err()).Warn("Interrupting cursor load due to cycle stop.")
			return it, nil
		}

		finished, uuid, err := uuidCollection.Next()
		i++

		if err != nil {
			log.WithError(err).Error("Failed to retrieve all elements from cursor!")
			return it, err
		}

		if finished {
			break
		}

		if i%10000 == 0 {
			end = time.Now()
			diff := end.Sub(start)
			log.WithField("collection", collection).WithField("duration", diff.String()).Infof("Loaded %v records", i)
			start = end
		}

		if i <= skip {
			continue
		}

		if strings.TrimSpace(uuid) == "" {
			blank++
			continue
		}

		if ok, err := blist(uuid); err != nil || ok {
			blacklisted++
			continue
		}

		it.append(uuid)
	}

	if b.s3ReadWriter != nil {
		err := persistInS3(b.s3ReadWriter, it)
		if err != nil {
			log.WithError(err).Warn("Failed to persist collection uuids to bucket")
		}
	}

	end = time.Now()
	diff := end.Sub(overallStart)

	log.WithField("collection", collection).WithField("duration", diff.String()).Infof("Finished loading %v records from DB", len(it.uuids))
	log.WithField("collection", collection).WithField("blacklisted", blacklisted).WithField("blank", blank).Info("Number of records blacklisted or blank.")

	return it, nil
}

func (i *InMemoryUUIDCollection) Next() (bool, string, error) {
	if i.Done() {
		return true, "", nil
	}
	return false, i.shift(), nil
}

func (i *InMemoryUUIDCollection) Length() int {
	return len(i.uuids) + i.skip
}

func (i *InMemoryUUIDCollection) Done() bool {
	return len(i.uuids) == 0
}

func (i *InMemoryUUIDCollection) Close() error {
	return nil
}

func (i *InMemoryUUIDCollection) append(uuid string) {
	i.uuids = append(i.uuids, uuid)
}

func (i *InMemoryUUIDCollection) shift() (x string) {
	x, i.uuids = i.uuids[0], i.uuids[1:]
	return
}
