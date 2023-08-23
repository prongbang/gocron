package database

import (
	"github.com/dgraph-io/badger/v3"
)

type Drivers interface {
	BadgerDB() *badger.DB
}

type drivers struct {
	Badger       *badger.DB
	BadgerDriver BadgerDBDriver
}

func (d *drivers) BadgerDB() *badger.DB {
	return d.Badger
}

func NewDrivers(
	badgerDriver BadgerDBDriver,
) Drivers {
	return &drivers{
		Badger:       badgerDriver.Connect(),
		BadgerDriver: badgerDriver,
	}
}
