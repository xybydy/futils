package main

import (
	"github.com/alecthomas/kong"
	"go.uber.org/zap"

	"github.com/xybydy/gdutils/gd"

	"github.com/xybydy/gdutils/logger"
)

type Global struct {
	Debug        bool `help:"Debug mode - creates log file."`
	Update       bool `short:"u" help:"Do not use local cache, force to obtain source folder information online"`
	NotTeamDrive bool `help:"If it is not a team drive link, you can add this parameter to improve interface query efficiency and reduce latency" short:"N"`
	// ServiceAccount bool `help:"Specify the service account for operation, provided that the json authorization file must be placed in the /sa Folder, please ensure that the SA account has Proper permissions。" short:"S" optional`
}

type CopyCmd struct {
	From string `arg name:"source id" help:"ID of source google folder"`
	To   string `arg name:"destination id" help:"ID of destination google folder"`
	Name string `help:"Rename the target folder, leave the original folder name blank" short:"n"`
	Size int64  `help:"If it is not a team drive link, you can add this parameter to improve interface query efficiency and reduce latency" short:"s"`
	DNCR bool   `short:"D" help:"do not create new root, Does not create a folder with the same name at the destination, will directly copy the files in the source folder to the destination folder as they are"`
	File bool   `help:"Copy a single file" short:"f"`
	Yes  bool   `help:"If a copy record is found, resume without asking" short:"y"`
}

func (c *CopyCmd) Run(g *Global) error {
	gd.InitApp()
	_, err := gd.Copy(c.From, c.To, c.Name, c.Size, g.Update, g.NotTeamDrive, c.DNCR)
	logger.Error("", err)
	return nil
}

type CountCmd struct {
	ID string `arg name:"Folder ID"`

	Sort   string `short:"s" help:"Sorting method of statistical results，Optional value name or size，If it is not filled in, it will be arranged in reverse order according to the number of files by default"`
	Type   string `short:"t" help:"The output type of the statistical result, the optional value is html/tree/snap/json/all, all means output the data as a json, it is best to use with -o. If not filled, the command line form will be output by default"`
	Output string `short:"o" help:"Statistics output file, suitable to use with -t'"`
}

func (c *CountCmd) Run(g *Global) error {
	gd.InitApp()
	err := gd.Count(c.ID, c.Sort, c.Type, c.Output, g.Update, g.NotTeamDrive)
	logger.Error("", err)
	return nil
}

func (c *CountCmd) Help() string {
	return `

Usage Examples: 
	- "gdutils count FOLDERID" 
			Get statistics of all files contained in https://drive.google.com/drive/folders/FOLDERID
	- "gdutils count FOLDERID -s size -t html -o out.html" 
			Get the personal drive root directory statistics, the results are output in HTML form, sorted in reverse order according to the total size, and saved to the out.html file in this directory (create new if it does not exist, overwrite if it exists)
	- "gdutils count FOLDERID -s name -t json -o out.json" 
			Get the statistics information of the root directory of the personal drive. The results are output in JSON format, sorted by file extension, and saved to the out.json file in this directory
	- "gdutils count FOLDERID -t all -o all.json" 
			Get the statistics of the root  Folder of the personal drive, output all file information (including folders) in JSON format, and save it to the all.json file in this Folder
`
}

type DeDupeCmd struct {
	ID string `arg name:"Folder ID"`

	Yes bool `help:"If duplicate items are found, delete them without asking" short:"y"`
}

func (c *DeDupeCmd) Run() error {
	return nil
}

type Md5Cmd struct {
	ID string `arg name:"Folder ID"`

	Size string `help:"Don't fill in the md5 records that store all files by default. If this value is set, files smaller than this size will be filtered out, which must end with b, such as 10mb" short:"s"`
}

func (c *Md5Cmd) Run() error {
	return nil
}

var Cli struct {
	Global

	Copy   CopyCmd   `cmd help:"Copy the files from one folder to another"`
	Count  CountCmd  `cmd`
	Dedupe DeDupeCmd `cmd`
	Md5    Md5Cmd    `cmd name:"md5"`
}

func main() {
	ctx := kong.Parse(&Cli,
		kong.Name("gdutils"),
		kong.Description(`Google Drive utilities`),
		kong.UsageOnError(),
	)

	err := ctx.Run(&Cli.Global)
	zap.S().Error(err)
	ctx.FatalIfErrorf(err)
}
