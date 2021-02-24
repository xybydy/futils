package summary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"google.golang.org/api/drive/v3"

	"github.com/xybydy/gdutils/database"
)

func putBetween(item []string, container string) string {
	retStrings := make([]string, 0)
	for _, i := range item {
		retStrings = append(retStrings, fmt.Sprintf("<%s>%s</%s>", container, i, container))
	}
	return strings.Join(retStrings, "")
}

func MakeHTML(s database.GdDBSummary) string {
	head := []string{"Type", "Number", "Size"}
	th := fmt.Sprintf("<tr>%s</tr>", putBetween(head, "th"))
	td := make([]string, 0)
	for _, v := range s.Details {
		data := []string{v.Ext, strconv.Itoa(v.Count), v.Size}
		inner := fmt.Sprintf("<tr>%s</tr>", putBetween(data, "td"))
		td = append(td, inner)
	}
	tail := fmt.Sprintf(`<tr style="font-weight:bold">%s</tr>`, putBetween([]string{"Total", strconv.Itoa(s.FileCount + s.FolderCount), s.TotalSize}, "td"))

	return fmt.Sprintf(`<table border="1" cellpadding="12" style="border-collapse:collapse;font-family:serif;font-size:22px;margin:10px auto;text-align: center">
    %s
    %s
    %s
  </table>`, th, strings.Join(td, ""), tail)
}

func MakeTable(s database.GdDBSummary) string {
	data := func() [][]string {
		ss := make([][]string, 0)
		for _, v := range s.Details {
			ss = append(ss, []string{v.Ext, strconv.Itoa(v.Count), v.Size})
		}
		return ss
	}()

	buf := new(bytes.Buffer)
	table := tablewriter.NewWriter(buf)
	table.SetHeader([]string{"Type", "Count", "Size"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
	)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetFooterAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	table.SetFooter([]string{"Total", strconv.Itoa(s.FileCount + s.FolderCount), s.TotalSize})

	table.SetFooterColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
	)
	for _, i := range data {
		table.Append(i)
	}
	table.Render()
	return buf.String()
}

func Summary(info []*drive.File, sortBy string) database.GdDBSummary {
	files := func() []*drive.File {
		var f []*drive.File
		for _, i := range info {
			if i.MimeType != "application/vnd.google-apps.folder" {
				f = append(f, i)
			}
		}
		return f
	}()
	fileCount := len(files)

	folders := func() []*drive.File {
		var f []*drive.File
		for _, i := range info {
			if i.MimeType == "application/vnd.google-apps.folder" {
				f = append(f, i)
			}
		}
		return f
	}()
	folderCount := len(folders)

	totalSize := func() string {
		size := int64(0)
		for _, i := range files {
			size += i.Size
		}
		return formatSize(float64(size))
	}()

	exts := make(map[string]int)
	sizes := make(map[string]int64)

	noExt := 0
	noExtSize := int64(0)

	for _, v := range files {
		ext := filepath.Ext(v.Name)
		if ext == "" || len(ext) > 10 {
			noExtSize += v.Size
			noExt++
			continue
		}
		exts[ext]++
		// if _, ok := exts[ext]; ok {
		// 	exts[ext]++
		// } else {
		// 	exts[ext] = 1
		// }
		sizes[ext] += v.Size
		// if _, ok := sizes[ext]; ok {
		// 	sizes[ext] += v.Size
		// } else {
		// 	sizes[ext] = v.Size
		// }
	}

	details := make([]database.SummaryItemInfo, 0)

	for k, v := range exts {
		size := sizes[k]
		detail := database.SummaryItemInfo{
			Ext:     k,
			Count:   v,
			Size:    formatSize(float64(size)),
			RawSize: size,
		}
		details = append(details, detail)
	}

	switch {
	case sortBy == "size":
		sort.Slice(details, func(i, j int) bool {
			return details[i].RawSize < details[j].RawSize
		})
	case sortBy == "name":
		sort.Slice(details, func(i, j int) bool {
			return details[i].Ext < details[j].Ext
		})
	default:
		sort.Slice(details, func(i, j int) bool {
			return details[i].Count < details[j].Count
		})
	}

	if noExt > 0 {
		detail := database.SummaryItemInfo{
			Ext:     "No Extension",
			Count:   noExt,
			Size:    formatSize(float64(noExtSize)),
			RawSize: noExtSize,
		}
		details = append(details, detail)
	}
	if folderCount > 0 {
		detail := database.SummaryItemInfo{
			Ext:   "Folder",
			Count: folderCount,
			Size:  "0",
		}
		details = append(details, detail)
	}

	return database.GdDBSummary{
		FileCount:   fileCount,
		FolderCount: folderCount,
		TotalSize:   totalSize,
		Details:     details,
	}
}

func formatSize(n float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	if n < 0 {
		return "invalid size"
	}
	flag := 0
	for n >= 1024 {
		n /= 1024
		flag++
	}
	return fmt.Sprintf("%.2f %s", n, units[flag])
}

func GetOutStr(info []*drive.File, outType, sort string) string {
	smy := Summary(info, sort)
	var outStr string
	switch outType {
	case "html":
		outStr = MakeHTML(smy)
	case "json":
		js, err := json.MarshalIndent(smy, "", "  ")
		if err != nil {
			log.Panic(err)
		}
		outStr = string(js)
	case "all":
		js, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Panic(err)
		}
		outStr = string(js)
	default:
		outStr = MakeTable(smy)
	}
	return outStr
}
