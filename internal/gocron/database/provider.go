package database

func NewDatabaseDriver() Drivers {
	badgerDriver := NewBadgerDB1Driver()
	databaseDrivers := NewDrivers(badgerDriver)
	return databaseDrivers
}
