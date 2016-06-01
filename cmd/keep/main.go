package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
	"github.com/docopt/docopt-go"
	"github.com/yml/keep"
)

var input string

func main() {

	usage := `keep password manager

Usage:
	keep read [options] <file> [--print]
	keep list [options] [<file>]
	keep add [options]

Options:
	-r --recipients=KEYS   List of key ids the message should be encypted 
	-d --dir=PATH          Account Directory
	-p --profile=NAME      Profile name
	-c --clipboard         Copy password to the clipboard
`

	args, err := docopt.Parse(usage, nil, true, "keep cli version: 0.0.1", false)
	if err != nil {
		fmt.Println("Dopopt specification cannot be parsed", err)
		os.Exit(1)
	}

	store, err := keep.LoadProfileStore()
	if err != nil {
		fmt.Println("An error occured while loading the profile store :", err)
		os.Exit(1)
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
			os.Exit(1)
		}
	}
	fmt.Println("Using profile : ", profile.Name)

	conf := keep.NewConfig(&profile)
	// Overriding the config with information from the cli parameters
	accountDir, ok := args["--dir"].(string)
	if ok {
		conf.AccountDir = accountDir
	}
	recipients, ok := args["--recipients"].(string)
	if ok {
		conf.RecipientKeyIds = recipients
	}

	if err != nil {
		fmt.Println("An error occured while reading the password", err)
		os.Exit(1)
	}

	if val, ok := args["read"]; ok == true && val == true {
		fmt.Println("Reading ...\n")
		fname, ok := args["<file>"].(string)
		if ok {
			fpath := filepath.Join(conf.AccountDir, fname)
			fmt.Println("file name:", fpath)
			account, err := keep.NewAccountFromFile(conf, fpath)
			if err != nil {
				fmt.Println("An error occured while creating and account from the clear text reader", err)
				os.Exit(1)
			}

			if account.IsSigned {
				fmt.Println("Credentials have been signed by :", account.SignedBy.PrivateKey.KeyIdShortString())
			} else {
				fmt.Println("\nWARNING: This credential is not signed !!!\n")
			}

			fmt.Println("Name : ", account.Name)
			fmt.Println("Username : ", account.Username)
			fmt.Println("Notes : ", account.Notes)
			if printOpt, ok := args["--print"]; ok && printOpt.(bool) == true {
				fmt.Println("Password : ", account.Password)
			}

			copyToclipboard := false
			if val, ok := args["-c"]; ok == true && val == true {
				copyToclipboard = true
			} else if val, ok := args["--clipboard"]; ok == true && val == true {
				copyToclipboard = true
			}
			if copyToclipboard {
				err = clipboard.WriteAll(account.Password)
				if err != nil {
					fmt.Println("An error occured while writing the password to the clipboard", err)
					os.Exit(1)
				}
			}
		}
	} else if val, ok := args["list"]; ok == true && val == true {
		fmt.Println("Listing ...\n")
		fileSubStr, ok := args["<file>"].(string)
		if !ok {
			fileSubStr = ""
		}

		files, err := conf.ListAccountFiles(fileSubStr)
		if err != nil {
			fmt.Printf("An error occured while listing the accounts", err)
			os.Exit(1)
		}
		for _, file := range files {
			fmt.Println(file.Name())
		}

	} else if val, ok := args["add"]; ok == true && val == true {
		fmt.Println("Adding ...\n")
		account, err := keep.NewAccountFromConsole(conf)
		if err != nil {
			fmt.Println("An error occured while retrieving account info from the console :", err)
			os.Exit(1)
		}

		content, err := account.Encrypt()
		if err != nil {
			fmt.Println("An error occured while encrypting the account to bytes", err)
			os.Exit(1)
		}

		fpath := filepath.Join(conf.AccountDir, account.Name)
		if _, err := os.Stat(fpath); !os.IsNotExist(err) {
			fmt.Printf("Account %s already exists\n", fpath)
			os.Exit(1)
		}
		fmt.Println("Writing file :", fpath)
		err = ioutil.WriteFile(fpath, content, 0600)
		if err != nil {
			fmt.Println("An error occured while writing the new account to disk", err)
			os.Exit(1)
		}
	}
	// fmt.Println(args, "\n", conf)
}
