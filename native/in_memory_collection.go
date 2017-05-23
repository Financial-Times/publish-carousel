package native

import (
	"strings"
	"time"

	"github.com/Financial-Times/publish-carousel/blacklist"
	log "github.com/Sirupsen/logrus"
)

type InMemoryUUIDCollection struct {
	uuids      []string
	collection string
}

func LoadIntoMemory(uuidCollection UUIDCollection, collection string, skip int, blist blacklist.IsBlacklisted) (UUIDCollection, error) {
	defer uuidCollection.Close()

	it := &InMemoryUUIDCollection{collection: collection, uuids: make([]string, 0)}
	if uuidCollection.Length() == 0 {
		log.WithField("collection", collection).Warn("No data in mongo cursor for this collection.")
		return it, nil
	}

	log.WithField("collection", collection).WithField("skip", skip).Info("Loading collection into memory...")

	i := 0
	overallStart := time.Now()
	start := time.Now()
	var end time.Time

	for {
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
			continue
		}

		if ok, err := blist(uuid); err != nil || ok {
			continue
		}

		it.append(uuid)
	}

	end = time.Now()
	diff := end.Sub(overallStart)
	log.WithField("collection", collection).WithField("duration", diff.String()).Infof("Finished loading %v records from DB", it.Length())

	return it, nil
}

func (i *InMemoryUUIDCollection) Next() (bool, string, error) {
	if i.Length() == 0 {
		return true, "", nil
	}
	return false, i.shift(), nil
}

func (i *InMemoryUUIDCollection) Length() int {
	return len(i.uuids)
}

func (i *InMemoryUUIDCollection) Done() bool {
	return i.Length() == 0
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
