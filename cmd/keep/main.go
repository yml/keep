package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/atotto/clipboard"
	"github.com/docopt/docopt-go"
	"github.com/yml/keep"
)

const (
	exitCodeOk    = 0
	exitCodeNotOk = 1
)

var input string

// printAndExitOnError exit the program with 1 as exit code after printing the message if the error is not nil.
func printAndExitOnError(err error, msg string) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(exitCodeNotOk)
	}
}

func printFileNames(files []os.FileInfo) {
	for i, file := range files {
		fmt.Printf("%d - %s\n", i, file.Name())
	}
}

func main() {

	usage := `keep password manager

Usage:
	keep read [options] <file> [<number>] [--print]
	keep list [options] [<file>]
	keep add [options]

Options:
	-r --recipients=KEYS   List of key ids the message should be encypted 
	-d --dir=PATH          Account Directory
	-p --profile=NAME      Profile name
	-c --clipboard         Copy password to the clipboard
`

	args, err := docopt.Parse(usage, nil, true, "keep cli version: 0.0.2", false)
	printAndExitOnError(err, "Docopt specification cannot be parsed")

	store, err := keep.LoadProfileStore()
	printAndExitOnError(err, "An error occured while loading the profile store")

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

	//fmt.Println(args, "\n", conf)
	if val, ok := args["read"]; ok == true && val == true {
		fmt.Println("Reading ...\n")
		fname, ok := args["<file>"].(string)
		if ok {
			accountPosition := -1
			files, err := conf.ListAccountFiles(fname)
			printAndExitOnError(err, "An error occured while gathering the accounts")
			snumber, ok := args["<number>"].(string)
			if ok {
				accountPosition, err = strconv.Atoi(snumber)
				printAndExitOnError(err, "An error occured while converting the <number> to an int")
			}
			switch l := len(files); {
			case l == 1:
				// Fall back to the name of the first match
				f := files[0]
				fname = f.Name()
			case l > 1 && -1 < accountPosition && accountPosition < l:
				f := files[accountPosition]
				fname = f.Name()
				fmt.Println("selecting filename :", fname)
			case l == 0:
				fmt.Println("No account name match :", fname)
				os.Exit(exitCodeNotOk)
			default:
				fmt.Println("There is more than one match")
				printFileNames(files)
				os.Exit(exitCodeNotOk)
			}

			fpath := filepath.Join(conf.AccountDir, fname)
			fmt.Println("file name:", fpath)
			account, err := keep.NewAccountFromFile(conf, fpath)
			if os.IsNotExist(err) {
				fmt.Printf("Account name (%s) does not exist.\n Listing ...\n\n", fname)
				os.Exit(exitCodeNotOk)
			}

			printAndExitOnError(err, "An error occured while creating and account from the clear text reader")

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
				printAndExitOnError(err, "An error occured while writing the password to the clipboard")
			}
		}
	} else if val, ok := args["list"]; ok == true && val == true {
		fmt.Println("Listing ...\n")
		fileSubStr, ok := args["<file>"].(string)
		if !ok {
			fileSubStr = ""
		}
		files, err := conf.ListAccountFiles(fileSubStr)
		printAndExitOnError(err, "An error occured while gathering the accounts")
		printFileNames(files)

	} else if val, ok := args["add"]; ok == true && val == true {
		fmt.Println("Adding ...\n")
		account, err := keep.NewAccountFromConsole(conf)
		printAndExitOnError(err, "An error occured while retrieving account info from the console :")

		content, err := account.Encrypt()
		printAndExitOnError(err, "An error occured while encrypting the account to bytes")

		fpath := filepath.Join(conf.AccountDir, account.Name)
		if _, err := os.Stat(fpath); !os.IsNotExist(err) {
			fmt.Printf("Account %s already exists\n", fpath)
			os.Exit(exitCodeNotOk)
		}
		fmt.Println("Writing file :", fpath)
		err = ioutil.WriteFile(fpath, content, 0600)
		printAndExitOnError(err, "An error occured while writing the new account to disk")
	}
}
