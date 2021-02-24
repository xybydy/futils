package database

import (
	"database/sql"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/xybydy/gdutils/utils"
)

type DriveDB struct {
	sync.Mutex
	db *sqlx.DB
}

func ConnectDB(name, path string) *DriveDB {
	var d = new(DriveDB)
	db, err := sqlx.Connect(name, path)
	utils.CheckErr(err)
	d.db = db
	return d
}

func (d *DriveDB) lock() {
	d.Lock()
}

func (d *DriveDB) unlock() {
	d.Unlock()
}

func (d *DriveDB) Exec(smt string, args ...interface{}) (sql.Result, error) {
	d.lock()
	res, err := d.db.Exec(smt, args...)
	d.unlock()

	return res, err
}

func (d *DriveDB) Get(dest interface{}, query string, args ...interface{}) error {
	d.lock()
	err := d.db.Get(dest, query, args...)
	d.unlock()

	return err
}

func (d *DriveDB) Select(dest interface{}, query string, args ...interface{}) error {
	d.lock()
	err := d.db.Select(dest, query, args...)
	d.unlock()

	return err
}

func (d *DriveDB) Close() error {
	return d.db.Close()
}
