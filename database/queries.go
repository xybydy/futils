package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/xybydy/gdutils/logger"
)

func (d *DriveDB) HashExist(id string) (bool, error) {
	_, err := d.Exec("SELECT * FROM hash WHERE gid = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *DriveDB) HashAdd(id, md5 string) error {
	_, err := d.Exec("INSERT INTO hash (gid, md5) VALUES (?, ?)", id, md5)
	return err
}

func (d *DriveDB) HashGetID(md5 string) (string, error) {
	var hash HashDB
	err := d.Get(&hash, "select * from hash where md5=? and status=? LIMIT=1", md5)
	return hash.Md5, err
}

func (d *DriveDB) GDExist(fid string) (bool, error) {
	_, err := d.Exec("SELECT fid FROM gd WHERE fid = ?", fid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *DriveDB) GDGet(fid string) (GdDB, bool, error) {
	record := GdDB{}
	err := d.Get(&record, "SELECT * FROM gd WHERE fid = ? LIMIT 1", fid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return record, false, nil
		}
		return record, false, err
	}
	return record, true, nil
}

func (d *DriveDB) GDUpdateItem(fid, info, subf string) error {
	logger.Debug("updating db for - %s", fid)
	_, err := d.Exec("UPDATE gd SET info=?, subf=?, mtime=? WHERE fid=?", info, subf, time.Now().Unix(), fid)
	return err
}

func (d *DriveDB) GDInsertItem(fid, info, subf string) error {
	logger.Debug("inserting db for - %s", fid)
	_, err := d.Exec("INSERT INTO gd (fid, info, subf, ctime) VALUES (?, ?, ?, ?)", fid, info, subf, time.Now().Unix())
	return err
}

func (d *DriveDB) GDUpdateSummary(fid, sum string) error {
	logger.Debug("updating summary for - %s", fid)
	_, err := d.Exec("UPDATE gd SET summary=?, mtime=? WHERE fid=?", sum, time.Now().Unix(), fid)
	return err
}

func (d *DriveDB) TaskGet(source, target string) (TaskDB, bool, error) {
	record := TaskDB{}
	err := d.Get(&record, "select id, status from task where source=? and target=?", source, target)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return record, false, nil
		}
		return record, false, err
	}
	return record, true, nil
}

func (d *DriveDB) TaskStatusUpdate(id int, status string) error {
	_, err := d.Exec("update task set status=?, ftime=? where id=?", status, time.Now().Unix(), id)
	return err
}

func (d *DriveDB) TaskUpdate(id int, status, rootMapping string) error {
	_, err := d.Exec("update task set status=?, mapping=? where id=?", status, rootMapping, id)
	return err
}

func (d *DriveDB) TaskAddMapping(taskID int, mappingRecord string) error {
	_, err := d.Exec("update task set mapping = mapping || ? where id=?", mappingRecord, taskID)
	return err
}

func (d *DriveDB) TaskInsert(source, target, status, rootMapping string) (sql.Result, error) {
	res, err := d.Exec("insert into task (source, target, status, mapping, ctime) values (?, ?, ?, ?, ?)", source, target, status, rootMapping, time.Now().Unix())
	return res, err
}

func (d *DriveDB) TaskDelete(source, target string) error {
	_, err := d.Exec("delete from task where source=? and target=?", source, target)
	return err
}

func (d *DriveDB) CopiedGet(taskID int) ([]CopiedDB, error) {
	var copied []CopiedDB
	logger.Debug("%s %d", "Checking copied table for id:", taskID)
	err := d.Select(&copied, "select fileid from copied where taskid=?", taskID)

	return copied, err
}

func (d *DriveDB) CopiedInsert(taskID int, fileId string) error {
	_, err := d.Exec("INSERT INTO copied (taskid, fileid) VALUES (?, ?)", taskID, fileId)
	return err
}

func (d *DriveDB) CopiedDelete(id int) error {
	_, err := d.Exec("delete from copied where taskid=?", id)
	return err
}
