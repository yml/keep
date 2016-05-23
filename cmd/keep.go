package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/docopt/docopt-go"
	"github.com/yml/keep"
)

var input string

func newCliConfig() (*keep.Config, error) {
	conf := keep.NewConfig()

	if conf.EncryptKeysIds == "" {
		fmt.Print("EncryptKeysIds: ")
		fmt.Scanln(&input)
		conf.EncryptKeysIds = input
	}
	return conf, nil

}

func main() {

	usage := `keep password manager

Usage:
	keep read [options] <file> [<dir>] [--print]
	keep list [options] [<dir>]
	keep add [options] [--prompt]

Options:
	-p --passphrase			Prompt a passphrase
	-e --encrypt-to-key-ids	List of key ids the message should be encypted time_colon
	-d --decrypt-to-key-ids	Private Key id to decrypt the message`

	args, err := docopt.Parse(usage, nil, true, "keep cli version: 0.0.1", false)
	if err != nil {
		log.Fatal("err: ", err)
	}

	conf, err := newCliConfig()
	if err != nil {
		fmt.Println("An error occured while reading the password", err)
		os.Exit(1)
	}

	if val, ok := args["read"]; ok == true && val == true {
		fmt.Println("\nReading ...")
		fname, ok := args["<file>"].(string)
		if ok {
			fmt.Println("file name:", fname)
			clearTextReader, err := conf.DecodeFile(filepath.Join(conf.PasswordDir, fname))
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
		fmt.Println("\nListing ...")
	} else if val, ok := args["add"]; ok == true && val == true {
		fmt.Println("\nAdding ...")
	}

	fmt.Println("\n\n* DEBUG ****************")
	fmt.Println(args, "\n", conf)
	fmt.Println("* DEBUG ****************")
}
