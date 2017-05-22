// Package main provides ...
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/atotto/clipboard"
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

	statusBar := tui.NewStatusBar("")
	statusBox := tui.NewVBox(statusBar)
	statusBox.SetTitle("Status")
	statusBox.SetBorder(true)

	statusBar.SetText(fmt.Sprintf("Using profile : %s", profile.Name))

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

	copyPasswordBtn := tui.NewButton("[ Copy ]")
	copyPasswordBtn.OnActivated(func(b *tui.Button) {
		// Grab the original clipboard value before changing it
		originalClipboard, err := clipboard.ReadAll()
		if err != nil {
			statusBar.SetText(fmt.Sprintf("Error: Could not copy from clipboard : %s", err))
		}
		err = clipboard.WriteAll(currentAcct.Password)
		if err != nil {
			statusBar.SetText(fmt.Sprintf("Error: Could not paste to clipboard : %s", err))
		}
		go func(s string) {
			time.Sleep(15 * time.Second)
			err = clipboard.WriteAll(s)
			statusBar.SetText(fmt.Sprintf("Error: Could not restore the clipboard: %s", err))
		}(originalClipboard)
	})

	accountDetail := tui.NewGrid(0, 0)
	accountDetail.AppendRow(tui.NewLabel("Username: "), username)
	accountDetail.AppendRow(tui.NewLabel("Notes: "), notes)
	accountDetail.AppendRow(tui.NewLabel(" Password: "), password, showPasswordBtn, copyPasswordBtn)

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
		accounts := fetchAccounts(conf, filter)
		accountList.AddItems(accounts...)
		accountList.SetSelected(0)
		if len(accounts) == 0 {
			statusBar.SetText("No account matching: " + filter)
		}
	})

	filterBox := tui.NewVBox(filterEntry)
	filterBox.SetTitle("Search an account")
	filterBox.SetBorder(true)

	tui.DefaultFocusChain.Set(filterEntry, accountList, showPasswordBtn, copyPasswordBtn)
	listSreen := tui.NewVBox(filterBox, accountBox, statusBox)
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
