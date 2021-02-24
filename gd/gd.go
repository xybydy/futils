package gd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/ratelimit"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	"github.com/xybydy/gdutils/auth"
	"github.com/xybydy/gdutils/config"
	"github.com/xybydy/gdutils/counter"
	"github.com/xybydy/gdutils/database"
	"github.com/xybydy/gdutils/logger"
	"github.com/xybydy/gdutils/prompter"
	"github.com/xybydy/gdutils/semaphore"
	"github.com/xybydy/gdutils/status"
	"github.com/xybydy/gdutils/summary"
	"github.com/xybydy/gdutils/utils"
)

// TODO: Proxy Support
// TODO: Handle Exit

// Queries per day 1,000,000,000
// Queries per 100 seconds per user 1,000
// Queries per 100 seconds 10,000

type ListArgs struct {
	Fields                    []googleapi.Field
	SortOrder                 string
	Query                     string
	includeItemsFromAllDrives bool
	supportsAllDrives         bool
}

func (l ListArgs) String() string {
	return fmt.Sprintf("fields: %v, sortOrder: %s, query: %s, includeItemsFromAllDrives: %t, supportsAllDrives: %t", l.Fields, l.SortOrder, l.Query, l.includeItemsFromAllDrives, l.supportsAllDrives)
}

const (
	FolderType = "application/vnd.google-apps.folder"
)

var (
	SaConfigs = new(auth.SaFileOrganizer)
	db        *database.DriveDB
	sema      = semaphore.New(config.ParallelLimit)
)

func InitApp() {
	logger.Debug("Connecting to db: %s", config.DBPath)
	db = database.ConnectDB("sqlite3", config.DBPath)
	logger.Debug("", "Service accounts initilization started")
	err := SaConfigs.InitFiles(config.SaLocation)
	utils.CheckErr(err)
}

func filterAll(arr []*drive.File, minSize int) ([]*drive.File, []*drive.File) {
	files := filterFiles(arr, minSize)
	folders := filterFolders(arr)
	return files, folders
}
func filterFiles(arr []*drive.File, minSize int) []*drive.File {
	files := make([]*drive.File, 0)
	for _, i := range arr {
		if i.MimeType != FolderType {
			if minSize > 0 {
				if i.Size >= int64(minSize) {
					files = append(files, i)
				}
			} else {
				files = append(files, i)
			}
		}
	}
	return files
}
func filterFolders(arr []*drive.File) []*drive.File {
	folders := make([]*drive.File, 0)
	for _, i := range arr {
		if i.MimeType == FolderType {
			folders = append(folders, i)
		}
	}
	return folders
}

func saveMd5(ctx context.Context, fid string, size int64, notTeamdrive bool, update bool) {
	logger.Debug("", "starting saving md5 hashes")
	var count counter.Counter
	f, err := walkAndSave(ctx, fid, notTeamdrive, update, false)
	utils.CheckErr(err)

	for _, i := range f {
		if i.MimeType != FolderType && i.Size >= size {
			if i.Md5Checksum == "" {
				continue
			}

			exists, err := db.HashExist(i.Id)
			if err != nil {
				logger.Error("", err)
			}
			if !exists {
				continue
			}

			err = db.HashAdd(i.Id, i.Md5Checksum)
			if err != nil {
				logger.Error("", err)
			}
			count.Inc()
		}
	}
	logger.Debug("%d no of hashes recorded", count.Get())
}

func validateFid(fid string) bool {
	logger.Debug("%s - %s", "validating fid", fid)

	whiteList := []string{"root", "appDataFolder", "photos"}
	if fid == "" {
		return false
	}
	for _, i := range whiteList {
		if i == fid {
			return true
		}
	}
	if len(fid) < 10 || len(fid) > 100 {
		return false
	}

	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, fid)
	logger.Error("", err)
	return matched
}

func getGIDByMd5(md5 string) string {
	gid, err := db.HashGetID(md5)
	logger.Error("", err)
	return gid
}

func getDriveName(ctx context.Context, fid string) (string, error) {
	logger.Debug("%s - %s", "getting drive name", fid)
	f, err := driveCall(ctx, fid)
	utils.CheckErr(err)
	return f.Name, nil
}

func saveFilesToDB(fid string, files []*drive.File) {
	logger.Debug("", "saving file infos to DB")
	var subf []string
	for _, i := range files {
		if i.MimeType == FolderType {
			subf = append(subf, i.Id)
		}
	}

	filesJSON, err := json.Marshal(files)
	if err != nil {
		logger.Error("", err)
	}
	if string(filesJSON) == "null" {
		filesJSON = []byte("[]")
	}

	var subfJSON []byte
	if len(subf) > 0 {
		subfJSON, err = json.Marshal(subf)
		if err != nil {
			logger.Error("", err)
		}
	} else {
		subfJSON = []byte("[]")
	}

	exists, err := db.GDExist(fid)
	if err != nil {
		logger.Error("", err)
		return
	}

	if exists {
		err := db.GDUpdateItem(fid, string(filesJSON), string(subfJSON))
		if err != nil {
			logger.Error("", err)
		}
	} else {
		err := db.GDInsertItem(fid, string(filesJSON), string(subfJSON))
		if err != nil {
			logger.Error("", err)
		}
	}
}

func walkAndSave(ctx context.Context, fid string, notTeamdrive, update, withModified bool) ([]*drive.File, error) {
	var resultMutex sync.Mutex
	var result []*drive.File
	var resultCount = new(counter.Counter)
	var recur func()
	now := time.Now()
	jobs := make(chan string, 10)
	wg := new(sync.WaitGroup)
	var pendingCount = new(counter.Counter)
	limiter := ratelimit.New(100)
	ctx, cancel := context.WithCancel(ctx)

	logger.Debug("%s: %s", "Walking the directory", fid)

	if update {
		logger.Debug("", "Updating the existing db records")
		exist, err := db.GDExist(fid)
		if err != nil {
			defer cancel()
			return nil, err
		}
		if exist {
			logger.Debug("", "Record found on db")
			logger.Debug("", "Updating the record")
			err = db.GDUpdateSummary(fid, "")
			if err != nil {
				logger.Error("", err)
			}
		}
	}

	go status.PrintStatus(ctx, pendingCount, resultCount, status.StatusReadPath)

	recur = func() {
		var shouldSave bool
		var files []*drive.File
		var err error

		defer wg.Done()
		defer func() {
			pendingCount.Dec()
		}()

		parent := <-jobs

		if update {
			limiter.Take()
			files, err = lsFolder(ctx, parent, notTeamdrive, withModified)
			utils.CheckErr(err)
			shouldSave = true
		} else {
			logger.Debug("Getting '%s' from db", parent)
			record, exists, err := db.GDGet(parent)
			if err != nil {
				logger.Error("", err)
			}
			if exists {
				logger.Debug("%s found on the db", parent)
				err = json.Unmarshal([]byte(record.Info.String), &files)
				utils.CheckErr(err)
			} else {
				logger.Debug("%s NOT found on the db", parent)
				files, err = lsFolder(ctx, parent, notTeamdrive, withModified)
				utils.CheckErr(err)
				shouldSave = true
			}
		}

		if shouldSave {
			saveFilesToDB(parent, files)
		}

		folders := make(chan *drive.File, len(files))
		for _, j := range files {
			if j.MimeType == FolderType {
				folders <- j
				pendingCount.Inc()
			}
		}
		close(folders)

		resultMutex.Lock()
		result = append(result, files...)
		resultCount.Add(int32(len(result)))
		resultMutex.Unlock()

		for i := range folders {
			wg.Add(1)
			jobs <- i.Id
			go recur()
		}
	}

	jobs <- fid
	wg.Add(1)
	recur()
	wg.Wait()
	cancel()

	smy := summary.Summary(result, "")
	if !smy.IsEmpty() {
		err := db.GDUpdateSummary(fid, smy.String())
		if err != nil {
			logger.Error("", err)
		}
	}

	logger.Info("Walking directory time took: %v", time.Since(now))
	logger.Info("Result no: %d", len(result))
	return result, nil
}

func getNameByID(ctx context.Context, fid string) (string, error) {
	logger.Debug("", "Getting name by id")
	info, err := getInfoByID(ctx, fid)
	if err != nil {
		logger.Error("", err)
		return "", err
	}
	switch {
	case info.Name != "":
		return info.Name, nil

	case info.Id == info.TeamDriveId:
		logger.Info("", "its a team drive")
		name, err := getDriveName(ctx, fid)
		if err != nil {
			logger.Error("", err)
			return "", err
		}
		return name, nil
	default:
		return fid, nil
	}
}

func getInfoByID(ctx context.Context, fid string) (*drive.File, error) {
	args := ListArgs{}
	args.Fields = []googleapi.Field{"id", "name", "teamDriveId", "md5Checksum", "mimeType", "size", "parents"}

	logger.Debug("Getting file info by id - %s", fid)
	f, err := fileGetCall(ctx, fid, args)
	utils.CheckErr(err)
	return f, nil
}

func getAllByFid(fid string) ([]*drive.File, error) {
	logger.Debug("", "Getting all from id for", fid)
	var recur func([]*drive.File, database.GdSubf) ([]*drive.File, error)
	recur = func(result []*drive.File, subf database.GdSubf) ([]*drive.File, error) {
		type res struct {
			Info []*drive.File
			Subf database.GdSubf
		}
		var ress []res
		var subSubf database.GdSubf
		if len(subf) == 0 {
			return result, nil
		}

		for i := range subf {
			var row database.GdDB
			var info []*drive.File
			var innerSubf database.GdSubf

			row, exists, err := db.GDGet(subf[i])
			// err := db.Get(&row, "SELECT * FROM gd WHERE fid = ? LIMIT 1", subf[i])
			if err != nil {
				logger.Error("", err)
			}
			if !exists {
				return result, sql.ErrNoRows
			}

			err = json.Unmarshal([]byte(row.Info.String), &info)
			utils.CheckErr(err)
			for j := range info {
				info[j].Parents[0] = subf[i]
			}

			if row.Subf.Valid {
				err = json.Unmarshal([]byte(row.Subf.String), &innerSubf)
				utils.CheckErr(err)
			}

			ress = append(ress, res{info, innerSubf})
		}

		for i := range ress {
			subSubf = append(subSubf, ress[i].Subf...)
			result = append(result, ress[i].Info...)
		}

		return recur(result, subSubf)
	}

	var result []*drive.File
	var subf database.GdSubf
	gd, exists, err := db.GDGet(fid)
	utils.CheckErr(err)

	if !exists {
		return []*drive.File{}, err
	}

	err = json.Unmarshal([]byte(gd.Info.String), &result)
	utils.CheckErr(err)

	for i := range result {
		result[i].Parents[0] = fid
	}

	if gd.Subf.Valid {
		err = json.Unmarshal([]byte(gd.Subf.String), &subf)
		utils.CheckErr(err)
	}

	if len(subf) == 0 {
		return result, nil
	}

	return recur(result, subf)
}

func createFolder(ctx context.Context, name string, parent []string) (*drive.File, error) {
	file := drive.File{Name: name, MimeType: FolderType, Parents: parent}
	args := ListArgs{}
	args.supportsAllDrives = true

	f, err := fileCreateCall(ctx, &file, args)
	utils.CheckErr(err)
	return f, err
}

func lsFolder(ctx context.Context, fid string, notTeamdrive, withModifiedtime bool) ([]*drive.File, error) {
	args := ListArgs{}

	if !(fid == "root" || notTeamdrive) {
		args.includeItemsFromAllDrives = true
		args.supportsAllDrives = true
	}

	args.SortOrder = "folder,name desc"
	args.Query = fmt.Sprintf("'%s' in parents and trashed = false", fid)
	args.Fields = []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,parents)"}

	if withModifiedtime {
		args.Fields = []googleapi.Field{"nextPageToken", "files(id,name,md5Checksum,mimeType,size,modifiedTime,parents)"}
	}
	files, err := fileListCall(ctx, args)

	return files, err
}

func Count(fid, sort, outType, output string, update, notTeamdrive bool) error {
	sort = strings.ToLower(sort)
	outType = strings.ToLower(outType)
	output = strings.ToLower(output)
	var outStr string
	var ctx = context.TODO()

	if !update {
		if outType != "" && sort != "" && output != "" {
			record, _, err := db.GDGet(fid)
			utils.CheckErr(err)

			if record.ContainsSummary() {
				fmt.Println(summary.MakeTable(record.GetSummary()))
				return nil
			}
		}

		var info, err = getAllByFid(fid)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Panic("", err)
		}
		if len(info) > 0 {
			outStr = summary.GetOutStr(info, outType, sort)
			if output != "" {
				err := ioutil.WriteFile(output, []byte(outStr), 0666)
				utils.CheckErr(err)
				return nil
			}
		}
	}
	files, err := walkAndSave(ctx, fid, notTeamdrive, update, false)
	utils.CheckErr(err)
	out := summary.GetOutStr(files, outType, sort)
	fmt.Println(out)
	return nil
}

func Copy(source, target, name string, minSize int64, update bool, notTeamdrive, dncr bool) (*drive.File, error) {
	logger.Debugw("Copy operation started", "source", source, "name", name, "minSize", minSize, "update", update, "notTeamdrive", notTeamdrive, "dncr", dncr)
	var task database.TaskDB
	var ctx = context.TODO()

	if target == "" {
		target = config.DefaultTarget
	}

	if target == "" {
		logger.Panic("", "Destination ID cannot be empty")
	}

	file, err := getInfoByID(ctx, source)
	if err != nil {
		return nil, err
	}
	if file.Id == "" {
		logger.Panic("Unable to access the link, please check if the link is valid and SA has the appropriate permissions：https://drive.google.com/drive/folders/%s", source)
	}
	if file.Id != "" && file.MimeType != FolderType {
		logger.Debug("Source is a file")
		f, err := copyFile(ctx, source, target, 0)
		if err == nil {
			return f, nil
		}
		logger.Error("", err)
	}

	task, exists, err := db.TaskGet(source, target)
	// if exists && task.Status == "copying" {
	// logger.Debug("", "This Task is already running. Force Quit")
	// return nil, err
	// }

	_, err = realCopy(ctx, source, target, name, int(minSize), update, dncr, notTeamdrive)
	if err != nil {
		logger.Error("Error copying folder %s", err)
		task, exists, err = db.TaskGet(source, target)
		if err != nil {
			logger.Error("", err)
		}
		if exists {
			err := db.TaskStatusUpdate(task.ID, "error")
			if err != nil {
				logger.Error("", err)
			}
		}
	}
	return nil, nil
}

func realCopy(ctx context.Context, source, target, name string, minSize int, update, dncr, notTeamdrive bool) (database.CopiedDB, error) {
	getNewRoot := func() (*drive.File, error) {
		if dncr {
			return &drive.File{Id: target}, nil
		}
		if name != "" {
			return createFolder(ctx, name, []string{target})
		}
		file, err := getNameByID(ctx, source)
		if err != nil {
			return nil, err
		}
		if file == "" {
			logger.Panic("Unable to access the link, please check if the link is valid and SA has the appropriate permissions：https://drive.google.com/drive/folders/%s", source)
		}
		return createFolder(ctx, file, []string{target})
	}

	logger.Debug("Checking source: %s - target:%s on TasksDB", source, target)
	task, exists, err := db.TaskGet(source, target)
	if !exists {
		logger.Debug("No record found on TaskDB source: %s - target:%s on TasksDB", source, target)
		newRoot, err := getNewRoot()
		utils.CheckErr(err)
		rootMapping := fmt.Sprintf("%s %s\n", source, newRoot.Id)
		res, err := db.TaskInsert(source, target, "copying", rootMapping)
		if err != nil {
			logger.Error("", err)
		}
		lastInsertID, err := res.LastInsertId()
		if err != nil {
			logger.Error("", err)
		}
		arr, err := walkAndSave(ctx, source, notTeamdrive, update, false)
		utils.CheckErr(err)

		files, folders := filterAll(arr, minSize)
		logger.Debug("Number of folders to be copied - %d", len(folders))
		logger.Debug("Number of files to be copied - %d", len(files))

		mapping := CreateFolders(ctx, source, nil, folders, newRoot, int(lastInsertID))
		copyFiles(ctx, files, mapping, newRoot, int(lastInsertID))
		err = db.TaskStatusUpdate(int(lastInsertID), "finished")
		if err != nil {
			logger.Error("", err)
		}

		return database.CopiedDB{
			TaskID: int(lastInsertID),
			FileID: newRoot.Id,
		}, nil
	} else if err == nil {
		copied, err := db.CopiedGet(task.ID)
		if err != nil {
			logger.Error("", err)
		}

		choice, _, err := prompter.PromptUserChoice.Run()
		if err != nil {
			logger.Error("", err)
		}

		switch {
		case choice == prompter.OptionExit:
			logger.Debug("", "Exit option selected")
			return database.CopiedDB{}, nil
		case choice == prompter.OptionContinue:
			logger.Debug("", "Continue option selected")
			var mapping [][]*drive.File
			copiedIds := make(map[string]bool)
			oldMappings := make(map[string]*drive.File)
			for _, i := range copied {
				copiedIds[i.FileID] = true
			}
			logger.Debug("", "Getting mapping from tasks db")
			mappingArray := strings.Split(strings.Trim(task.Mapping, " "), "\n")
			for _, i := range mappingArray {
				var keh []*drive.File
				maps := strings.Split(i, " ")
				for _, j := range maps {
					keh = append(keh, &drive.File{Id: j})
				}
				mapping = append(mapping, keh)
			}

			root := mapping[0][1]
			for _, i := range mapping {
				oldMappings[i[0].Id] = i[1]
			}
			logger.Debug("%s - %s", "updating db", task.ID)
			err := db.TaskStatusUpdate(task.ID, "copying")
			if err != nil {
				logger.Error("", err)
			}
			arr, err := walkAndSave(ctx, source, notTeamdrive, update, false)
			if err != nil {
				logger.Error("", err)
			}
			files, folders := filterAll(arr, minSize)
			logger.Debug("No of files: %d", len(files))
			logger.Debug("No of folders: %d", len(folders))

			allMapping := CreateFolders(ctx, source, oldMappings, folders, root, task.ID)
			copyFiles(ctx, files, allMapping, root, task.ID)
			err = db.TaskStatusUpdate(task.ID, "finished")
			if err != nil {
				logger.Error("", err)
			}

			return database.CopiedDB{
				TaskID: task.ID,
				FileID: root.Id,
			}, nil
		case choice == prompter.OptionRestart:
			logger.Debug("", "Getting root folder")
			newRoot, err := getNewRoot()
			utils.CheckErr(err)
			rootMapping := fmt.Sprintf("%s %s\n", source, newRoot.Id)
			err = db.TaskUpdate(task.ID, "copying", rootMapping)
			if err != nil {
				logger.Error("", err)
			}
			err = db.CopiedDelete(task.ID)
			if err != nil {
				logger.Error("", err)
			}
			arr, err := walkAndSave(ctx, source, notTeamdrive, update, false)
			utils.CheckErr(err)
			files, folders := filterAll(arr, minSize)
			fmt.Println("Number of folders to be copied", len(folders))
			fmt.Println("Number of files to be copied", len(files))
			mapping := CreateFolders(ctx, source, nil, folders, newRoot, task.ID)
			copyFiles(ctx, files, mapping, newRoot, task.ID)
			err = db.TaskStatusUpdate(task.ID, "finished")
			if err != nil {
				logger.Error("", err)
			}
			return database.CopiedDB{
				TaskID: task.ID,
				FileID: newRoot.Id,
			}, nil
		default:
			fmt.Println("Exit")
			return database.CopiedDB{}, nil
		}
	}
	logger.Error("", err.Error())
	return database.CopiedDB{}, err
}

func copyFiles(ctx context.Context, files []*drive.File, mapping map[string]*drive.File, root *drive.File, taskID int) {
	var wg sync.WaitGroup
	var count = new(counter.Counter)
	var pendingCount = new(counter.Counter)
	ctx, cancel := context.WithCancel(ctx)
	limiter := ratelimit.New(100)
	defer cancel()

	if len(files) == 0 {
		return
	}
	fmt.Printf("\nStarted copying files, total：%d\n", len(files))
	logger.Info("Started copying files, total：%d", len(files))
	go status.PrintStatus(ctx, pendingCount, count, status.StatusCopy)

	pendingCount.Set(int32(len(files)))
	for _, item := range files {
		wg.Add(1)
		go func(innerItem *drive.File) {
			sema.Wait()
			defer wg.Done()
			defer func() {
				sema.Signal()
			}()

			var target *drive.File
			if innerItem.Id == "" {
				return
			}

			if len(innerItem.Parents) > 0 && mapping[innerItem.Parents[0]].Id != "" {
				target = mapping[innerItem.Parents[0]]
			} else {
				target = root
			}

			limiter.Take()
			newfile, err := copyFile(ctx, innerItem.Id, target.Id, taskID)
			pendingCount.Dec()
			// todo buraya db'ye hata olarak ekleme ozelligi konulacak
			if err != nil {
				logger.Error("FAA %s", err)
				return
			}

			// utils.CheckErr(err)

			if newfile.Id != "" {
				count.Inc()
				err := db.CopiedInsert(taskID, innerItem.Id)
				if err != nil {
					logger.Error("", err)
				}
			}
		}(item)
	}
	wg.Wait()
	cancel()
}

func copyFile(ctx context.Context, id, parent string, taskID int) (*drive.File, error) {
	args := ListArgs{supportsAllDrives: true}
	file, err := fileCopyCall(ctx, id, parent, args)
	if err != nil {
		if taskID != 0 {
			err := db.TaskUpdate(taskID, "error", "")
			logger.Error("", err)
		}
		return nil, err
	}
	return file, err
}

func CreateFolders(ctx context.Context, source string, oldMapping map[string]*drive.File, folders []*drive.File, root *drive.File, taskId int) map[string]*drive.File {
	logger.Debugw("Creating folders", "source", source, "oldMapping", oldMapping, "folders", folders)
	var wg sync.WaitGroup
	var mut = new(sync.Mutex)
	var count = new(counter.Counter)
	var pendingCount = new(counter.Counter)
	var sameLevels []*drive.File
	limiter := ratelimit.New(100)
	var sameLevelsMissed []*drive.File
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mapping := make(map[string]*drive.File)
	if len(oldMapping) > 0 {
		mapping = oldMapping
	}
	mapping[source] = root
	if len(folders) == 0 {
		return mapping
	}

	missedFolders := make([]*drive.File, 0)
	for _, i := range folders {
		if v, ok := mapping[i.Id]; !ok {
			missedFolders = append(missedFolders, v)
		}
	}

	fmt.Printf("Start creating folders, total: %d\n", len(missedFolders))

	go status.PrintStatus(ctx, pendingCount, count, status.StatusCreateFolder)

	for _, i := range folders {
		if i.Parents[0] == folders[0].Parents[0] {
			sameLevels = append(sameLevels, i)
		}
	}

	pendingCount.Set(int32(len(folders)))

	for len(sameLevels) > 0 {
		var lolo []*drive.File
		for _, i := range sameLevels {
			if _, ok := mapping[i.Id]; !ok {
				lolo = append(lolo, i)
			}
		}
		sameLevelsMissed = lolo
		for _, item := range sameLevelsMissed {
			wg.Add(1)
			go func(innerItem *drive.File) {
				sema.Wait()
				defer wg.Done()
				var target string
				mut.Lock()
				if _, ok := mapping[innerItem.Parents[0]]; !ok {
					target = root.Id
				} else {
					target = mapping[innerItem.Parents[0]].Id
				}
				mut.Unlock()
				limiter.Take()
				newFolder, err := createFolder(ctx, innerItem.Name, []string{target})
				utils.CheckErr(err)
				count.Inc()
				pendingCount.Dec()
				mut.Lock()
				mapping[innerItem.Id] = newFolder
				mut.Unlock()
				mappingRecord := fmt.Sprintf("%s %s\n", innerItem.Id, newFolder.Id)
				err = db.TaskAddMapping(taskId, mappingRecord)
				if err != nil {
					logger.Error("", err)
				}
				sema.Signal()
			}(item)
		}
		wg.Wait()
		var k []*drive.File
		for _, i := range sameLevels {
			for _, j := range folders {
				if i.Id == j.Parents[0] {
					k = append(k, j)
				}
			}
		}
		sameLevels = k
	}
	return mapping
}
