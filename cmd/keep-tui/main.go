// Package main provides ...
package main

import (
	"fmt"
	"os"

	docopt "github.com/docopt/docopt-go"
	tui "github.com/marcusolsson/tui-go"
	"github.com/yml/keep"
)

var (
	filter      = ""
	currentAcct = &keep.Account{}
)

const (
	exitCodeOk     = 0
	exitCodeNotOk  = 1
	hiddenPassword = "*************"
)

func main() {

	usage := `keep-ui is a terminal user interface for keep
Usage:
	keep-ui [options]

Options:
	-p --profile=NAME      Profile name
`

	args, err := docopt.Parse(usage, nil, true, "keep cli version: 0.2", false)
	if err != nil {
		fmt.Println("Docopt specification cannot be parsed")
		os.Exit(exitCodeNotOk)
	}

	store, err := keep.LoadProfileStore()
	if err != nil {
		panic(err)
	}

	// defaulting to the first profile
	profile := store[0]
	profileName, ok := args["--profile"].(string)
	if ok {
		profileFound := false
		for _, p := range store {
			if profileName == p.Name {
				profile = p
				profileFound = true
				break
			}
		}
		if !profileFound {
			fmt.Printf("Profile (%s) not found\n", profileName)
			os.Exit(exitCodeNotOk)
		}
	}
	// TODO: writing the profile.Name in the status bar
	//fmt.Println("Using profile : ", profile.Name)

	conf := keep.NewConfig(&profile)

	// Setting up the interface
	username := tui.NewLabel("")
	notes := tui.NewLabel("")
	password := tui.NewLabel("")

	showPasswordState := false
	showPasswordBtn := tui.NewButton("[ show ]")
	showPasswordBtn.OnActivated(func(b *tui.Button) {
		if showPasswordState {
			password.SetText(hiddenPassword)
			showPasswordState = false
		} else {
			password.SetText(currentAcct.Password)
			showPasswordState = true
		}
	})

	accountDetail := tui.NewGrid(0, 0)
	accountDetail.AppendRow(tui.NewLabel("Username: "), username)
	accountDetail.AppendRow(tui.NewLabel("Notes: "), notes)
	accountDetail.AppendRow(tui.NewLabel(" Password: "), password, showPasswordBtn)

	accountDetailBox := tui.NewVBox(accountDetail)
	accountDetailBox.SetBorder(true)
	accountDetailBox.SetSizePolicy(tui.Preferred, tui.Preferred)

	accountList := tui.NewList()
	accountList.SetSelected(0)
	accountList.OnSelectionChanged(func(l *tui.List) {
		fname := accountList.SelectedItem()
		currentAcct = getAccount(conf, fname)
		username.SetText(currentAcct.Name)
		notes.SetText(currentAcct.Notes)
		password.SetText(hiddenPassword)
	})

	accountBox := tui.NewHBox(accountList, accountDetailBox)
	accountBox.SetTitle(" Accounts ")
	accountBox.SetBorder(true)
	accountBox.SetSizePolicy(tui.Expanding, tui.Expanding)

	filterEntry := tui.NewEntry()
	filterEntry.SetText(filter)
	filterEntry.OnSubmit(func(e *tui.Entry) {
		filter = e.Text()
		accountList.RemoveItems()
		accountList.AddItems(fetchAccounts(conf, filter)...)
		accountList.SetSelected(0)
	})

	filterBox := tui.NewVBox(filterEntry)
	filterBox.SetTitle("Search an account")
	filterBox.SetBorder(true)

	tui.DefaultFocusChain.Set(filterEntry, accountList, showPasswordBtn)
	listSreen := tui.NewVBox(filterBox, accountBox)
	ui := tui.New(listSreen)
	ui.SetKeybinding(tui.KeyEsc, func() { ui.Quit() })

	if err := ui.Run(); err != nil {
		panic(err)
	}
}

func fetchAccounts(conf *keep.Config, filter string) []string {
	files, err := conf.ListAccountFiles(filter)
	if err != nil {
		panic(err)
	}
	lst := make([]string, len(files))
	for i, f := range files {
		lst[i] = f.Name()
	}
	return lst
}

func getAccount(conf *keep.Config, fname string) *keep.Account {
	account, err := keep.NewAccountFromFile(conf, fname)
	if err != nil {
		fmt.Println("An error occured while getting the account: ", err)
	}
	return account
}
