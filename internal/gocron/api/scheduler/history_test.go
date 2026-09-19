package scheduler

import (
	"slices"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v3"
)

type memDrivers struct{ db *badger.DB }

func (m memDrivers) BadgerDB() *badger.DB { return m.db }

func TestHistory(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewRepository(memDrivers{db})

	if err := repo.Add("job-a", CreateScheduler{Job: "job-a", Cron: "* * * * *"}); err != nil {
		t.Fatal(err)
	}
	// Oldest → newest.
	runs := []History{
		{Job: "job-a", Project: "billing", URL: "http://x/invoice", Status: 200},
		{Job: "job-b", Project: "reports", URL: "http://x/daily", Status: 500, Response: "Boom"},
		{Job: "job-a", Project: "billing", URL: "http://x/invoice", Status: 404, Response: "connection refused"},
		{Job: "job-c", URL: "http://x/ping", Status: 204},
	}
	base := time.Now()
	for i, h := range runs {
		h.StartedAt = base.Add(time.Duration(i) * time.Second)
		if err := repo.AddHistory(h); err != nil {
			t.Fatal(err)
		}
	}

	if got := repo.GetConfigAll(); len(got) != 1 || got[0].Job != "job-a" {
		t.Fatalf("history leaked into job configs: %+v", got)
	}

	statuses := func(p HistoryPage) (s []int) {
		for _, h := range p.Items {
			s = append(s, h.Status)
		}
		return s
	}
	cases := []struct {
		name  string
		q     HistoryQuery
		total int
		want  []int
	}{
		{"all newest first", HistoryQuery{}, 4, []int{204, 404, 500, 200}},
		{"job", HistoryQuery{Job: "job-a"}, 2, []int{404, 200}},
		{"project", HistoryQuery{Project: "reports"}, 1, []int{500}},
		{"failed", HistoryQuery{Status: "failed"}, 2, []int{404, 500}},
		{"ok", HistoryQuery{Status: "ok"}, 2, []int{204, 200}},
		{"search response, case-insensitive", HistoryQuery{Q: "BOOM"}, 1, []int{500}},
		{"search url", HistoryQuery{Q: "invoice"}, 2, []int{404, 200}},
		{"page 2", HistoryQuery{Page: 2, Limit: 3}, 4, []int{200}},
		{"page past end", HistoryQuery{Page: 3, Limit: 3}, 4, nil},
	}
	for _, c := range cases {
		if c.q.Page == 0 {
			c.q.Page = 1
		}
		if c.q.Limit == 0 {
			c.q.Limit = 10
		}
		p := repo.GetHistory(c.q)
		if got := statuses(p); p.Total != c.total || !slices.Equal(got, c.want) {
			t.Errorf("%s: total=%d items=%v, want total=%d items=%v", c.name, p.Total, got, c.total, c.want)
		}
	}

	if p := repo.GetHistory(HistoryQuery{Page: 1, Limit: 10, Status: "failed"}); len(p.Projects) != 2 || p.Projects[0] != "billing" {
		t.Errorf("projects should list all history projects regardless of filters: %v", p.Projects)
	}
}
