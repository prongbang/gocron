package scheduler

import (
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
	base := time.Now()
	for i, job := range []string{"job-a", "job-b", "job-a"} {
		if err := repo.AddHistory(History{Job: job, Status: 200 + i, StartedAt: base.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	if got := repo.GetConfigAll(); len(got) != 1 || got[0].Job != "job-a" {
		t.Fatalf("history leaked into job configs: %+v", got)
	}
	all := repo.GetHistory("", 10)
	if len(all) != 3 || all[0].Status != 202 || all[2].Status != 200 {
		t.Fatalf("want 3 records newest first, got %+v", all)
	}
	if a := repo.GetHistory("job-a", 10); len(a) != 2 || a[0].Status != 202 {
		t.Fatalf("job filter: %+v", a)
	}
	if one := repo.GetHistory("", 1); len(one) != 1 || one[0].Status != 202 {
		t.Fatalf("limit: %+v", one)
	}
}
