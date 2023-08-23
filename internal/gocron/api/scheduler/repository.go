package scheduler

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v3"
	"github.com/prongbang/gocron/internal/gocron/database"
)

type Repository interface {
	GetConfigAll() []CreateScheduler
	Add(key string, data CreateScheduler) error
	Delete(key string) error
}

type repository struct {
	Drivers database.Drivers
}

func (r *repository) GetConfigAll() []CreateScheduler {
	con := r.Drivers.BadgerDB()

	data := []CreateScheduler{}

	// Find config all from database
	err := con.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		itr := txn.NewIterator(opts)
		defer itr.Close()

		for itr.Rewind(); itr.Valid(); itr.Next() {
			item := itr.Item()
			fn := func(v []byte) error {
				s := CreateScheduler{}
				if err := json.Unmarshal(v, &s); err == nil {
					data = append(data, s)
				}
				return nil
			}

			if err := item.Value(fn); err != nil {
				fmt.Println("[ERROR]", err)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println("[ERROR]", err)
	}

	return data
}

func (r *repository) Delete(key string) error {
	con := r.Drivers.BadgerDB()
	err := con.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	return err
}

func (r *repository) Add(key string, data CreateScheduler) error {
	con := r.Drivers.BadgerDB()
	value, em := json.Marshal(data)
	if em != nil {
		return em
	}
	eu := con.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
	return eu
}

func NewRepository(drivers database.Drivers) Repository {
	return &repository{
		Drivers: drivers,
	}
}
