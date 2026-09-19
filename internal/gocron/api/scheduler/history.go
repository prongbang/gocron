package scheduler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
)

// History is one execution of a job. It snapshots the job config so the
// record stays readable after the job is stopped.
type History struct {
	Job        string    `json:"job"`
	Project    string    `json:"project"`
	Cron       string    `json:"cron"`
	Method     string    `json:"method"`
	URL        string    `json:"url"`
	Status     int       `json:"status"`
	Response   string    `json:"response,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	DurationMs int64     `json:"duration_ms"`
}

var historyPrefix = []byte("history:")

// ponytail: fixed 7-day retention via badger TTL, make it configurable if someone needs longer.
const historyTTL = 7 * 24 * time.Hour

// Keys sort by start time, so reverse iteration yields newest first.
func historyKey(h History) []byte {
	return []byte(fmt.Sprintf("%s%020d:%s", historyPrefix, h.StartedAt.UnixNano(), h.Job))
}

func (r *repository) AddHistory(h History) error {
	value, err := json.Marshal(h)
	if err != nil {
		return err
	}
	return r.Drivers.BadgerDB().Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry(historyKey(h), value).WithTTL(historyTTL))
	})
}

// GetHistory returns up to limit records, newest first, optionally for one job.
func (r *repository) GetHistory(job string, limit int) []History {
	data := []History{}
	err := r.Drivers.BadgerDB().View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		opts.Prefix = historyPrefix
		itr := txn.NewIterator(opts)
		defer itr.Close()

		for itr.Seek(append(historyPrefix, 0xff)); itr.ValidForPrefix(historyPrefix) && len(data) < limit; itr.Next() {
			_ = itr.Item().Value(func(v []byte) error {
				h := History{}
				if json.Unmarshal(v, &h) == nil && (job == "" || h.Job == job) {
					data = append(data, h)
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		fmt.Println("[ERROR]", err)
	}
	return data
}
