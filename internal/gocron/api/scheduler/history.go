package scheduler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

type HistoryQuery struct {
	Job     string
	Project string
	Status  string // "", "ok" or "failed"
	Q       string // case-insensitive substring of job, project, method, url, status or response
	Page    int    // 1-based
	Limit   int
}

type HistoryPage struct {
	Items    []History `json:"items"`
	Total    int       `json:"total"`
	Projects []string  `json:"projects"` // every project seen in history, for the filter
}

func (q HistoryQuery) match(h History) bool {
	ok := h.Status >= 200 && h.Status < 300
	switch {
	case q.Job != "" && h.Job != q.Job,
		q.Project != "" && h.Project != q.Project,
		q.Status == "ok" && !ok,
		q.Status == "failed" && ok:
		return false
	}
	if q.Q == "" {
		return true
	}
	text := strings.ToLower(strings.Join([]string{h.Job, h.Project, h.Method, h.URL, strconv.Itoa(h.Status), h.Response}, " "))
	return strings.Contains(text, strings.ToLower(q.Q))
}

// GetHistory returns one page of matching records, newest first.
// ponytail: full scan per request (fine for 7 days of history), add secondary indexes if it gets slow.
func (r *repository) GetHistory(q HistoryQuery) HistoryPage {
	res := HistoryPage{Items: []History{}, Projects: []string{}}
	skip := (q.Page - 1) * q.Limit
	projects := map[string]bool{}
	err := r.Drivers.BadgerDB().View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		opts.Prefix = historyPrefix
		itr := txn.NewIterator(opts)
		defer itr.Close()

		for itr.Seek([]byte(string(historyPrefix) + "\xff")); itr.ValidForPrefix(historyPrefix); itr.Next() {
			_ = itr.Item().Value(func(v []byte) error {
				h := History{}
				if json.Unmarshal(v, &h) != nil {
					return nil
				}
				if h.Project != "" {
					projects[h.Project] = true
				}
				if q.match(h) {
					if res.Total >= skip && len(res.Items) < q.Limit {
						res.Items = append(res.Items, h)
					}
					res.Total++
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		fmt.Println("[ERROR]", err)
	}
	for p := range projects {
		res.Projects = append(res.Projects, p)
	}
	sort.Strings(res.Projects)
	return res
}
