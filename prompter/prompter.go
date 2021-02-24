package prompter

import (
	"github.com/manifoldco/promptui"
)

const (
	OptionContinue = iota
	OptionRestart
	OptionExit
	optionYes = 0
	optionNo  = 1
)

var selectTemplate = &promptui.SelectTemplates{
	Label:    `{{ . }}?`,
	Active:   "\U00002771 {{ .Name }} - {{ .Description | faint}}",
	Inactive: "  {{ .Name }} - {{ .Description | faint}}",
}

var userChoices = []struct {
	Name        string
	Description string
}{
	{"Continue", "Resume the transfer"},
	{"Restart", "Restart the process"},
	{"Exit", "Exit"},
}

var yesOrNo = []struct {
	Name        string
	Description string
}{
	{"Yes", "Confirm Deletion"},
	{"No", "Do not delete"},
}

var PromptUserChoice = promptui.Select{
	Label:        "Do you wish to continue",
	Items:        userChoices,
	HideHelp:     true,
	HideSelected: true,
	Templates:    selectTemplate,
}

var promptDuplicate = promptui.Select{
	Label:        `Duplicate files detected ${file_number}，Empty Folders detected${folder_number}，Delete them?`,
	Items:        yesOrNo,
	HideHelp:     true,
	HideSelected: true,
	Templates:    selectTemplate,
}
