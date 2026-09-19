package scheduler

import (
	"testing"

	"github.com/dgraph-io/badger/v3"
)

func TestGetConfigAllIgnoresNamespacedKeys(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewRepository(memDrivers{db})

	if err := repo.Add("5c964cf984484051a5ce10536c039a5a", CreateScheduler{Job: "5c964cf984484051a5ce10536c039a5a"}); err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(txn *badger.Txn) error {
		for _, k := range []string{"user:alice", "session:abc"} {
			if err := txn.Set([]byte(k), []byte(`{"job":"not-a-job"}`)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := repo.GetConfigAll(); len(got) != 1 {
		t.Fatalf("want only the real job, got %+v", got)
	}
}
