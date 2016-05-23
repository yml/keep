package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/yml/keep"
)

var input string

func main() {

	usage := `keep password manager

Usage:
	keep read [options] <file> [<dir>] [--print]
	keep list [options] [<file>]
	keep add [options] [--prompt]

Options:
	-r --recipients		List of key ids the message should be encypted time_colon
	-d --account-dir	Account Directory
`

	args, err := docopt.Parse(usage, nil, true, "keep cli version: 0.0.1", false)
	if err != nil {
		log.Fatal("err: ", err)
	}

	conf := keep.NewConfig()
	// Overriding the config with information from the cli parameters
	accountDir, ok := args["--account-dir"].(string)
	if ok {
		conf.AccountDir = accountDir
	}
	recipients, ok := args["--recipients"].(string)
	if ok {
		conf.RecipientKeysIds = recipients
	}

	if err != nil {
		fmt.Println("An error occured while reading the password", err)
		os.Exit(1)
	}

	if val, ok := args["read"]; ok == true && val == true {
		fmt.Println("Reading ...")
		fname, ok := args["<file>"].(string)
		if ok {
			fpath := filepath.Join(conf.AccountDir, fname)
			fmt.Println("file name:", fpath)
			clearTextReader, err := conf.DecodeFile(fpath)
			if err != nil {
				fmt.Println("An error occured while building the clear text reader", err)
				os.Exit(1)
			}
			content, err := ioutil.ReadAll(clearTextReader)
			if err != nil {
				fmt.Println("An error occured while reading the file", err)
				os.Exit(1)
			}

			account, err := keep.NewAccountFromString(fname, string(content))
			if err != nil {
				fmt.Println("An error occured while creating and account from file content", err)
				os.Exit(1)
			}

			fmt.Println("\n\n")
			fmt.Println("Name : ", account.Name)
			fmt.Println("Username : ", account.Username)
			fmt.Println("Notes : ", account.Notes)
			if printOpt, ok := args["--print"]; ok && printOpt.(bool) == true {
				fmt.Println("Password : ", account.Password)
			}
		}
	} else if val, ok := args["list"]; ok == true && val == true {
		fmt.Println("Listing ...\n")
		files, err := conf.ListFileInAccount()
		if err != nil {
			fmt.Printf("An error occured while listing the accounts", err)
			os.Exit(1)
		}

		fileSubStr, ok := args["<file>"].(string)
		if !ok {
			fileSubStr = ""
		}

		for _, file := range files {
			fname := file.Name()
			if strings.Contains(fname, fileSubStr) {
				fmt.Println(fname)
			}
		}

	} else if val, ok := args["add"]; ok == true && val == true {
		fmt.Println("Adding ...")
		panic("Not Implemented")
	}

	fmt.Println("\n\n* DEBUG ****************")
	fmt.Println(args, "\n", conf)
	fmt.Println("* DEBUG ****************")
}
