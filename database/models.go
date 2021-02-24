package database

import (
	"database/sql"
	"encoding/json"

	"google.golang.org/api/drive/v3"

	"github.com/xybydy/gdutils/utils"
)

type GdDB struct {
	ID      int
	Fid     sql.NullString
	Info    sql.NullString
	Summary sql.NullString
	Subf    sql.NullString
	Ctime   sql.NullInt64
	Mtime   sql.NullInt64
}

func (g GdDB) GetInfo() []*drive.File {
	var d []*drive.File
	err := json.Unmarshal([]byte(g.Info.String), &d)
	utils.CheckErr(err)
	return d
}

func (g GdDB) ContainsSummary() bool {
	return g.Summary.Valid
}

func (g GdDB) GetSummary() GdDBSummary {
	var d GdDBSummary
	err := json.Unmarshal([]byte(g.Summary.String), &d)
	utils.CheckErr(err)
	return d
}

type GdDBSummary struct {
	FileCount   int
	FolderCount int
	TotalSize   string
	Details     []SummaryItemInfo
}

func (g GdDBSummary) IsEmpty() bool {
	return g.FileCount == 0 && g.FolderCount == 0 && g.TotalSize == ""
}

func (g GdDBSummary) String() string {
	q, err := json.Marshal(g)
	utils.CheckErr(err)
	return string(q)
}

type SummaryItemInfo struct {
	Ext     string
	Count   int
	Size    string
	RawSize int64
}

type GdSubf []string

type TaskDB struct {
	ID      int
	Source  string
	Target  string
	Status  string
	Copied  string
	Mapping string
	Ctime   sql.NullInt64
	Ftime   sql.NullInt64
}

type CopiedDB struct {
	TaskID int
	FileID string
}

type BookmarkDB struct {
	Alias  string
	Target string
}

type HashDB struct {
	Md5    string
	GID    string
	Status string
}

type ItemInfo struct {
	ID           string
	MimeType     string `json:"mimeType"`
	ModifiedTime string `json:"modifiedTime"`
	Name         string
	Parents      []string
}
