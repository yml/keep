package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/docopt/docopt-go"
	"github.com/yml/geep"
)

var input string

func newCliConfig() (*geep.Config, error) {
	conf := geep.NewConfig()

	if conf.DecryptKeyIds == "" {
		fmt.Print("DecryptKeyIds: ")
		fmt.Scanln(&input)
		conf.DecryptKeyIds = input
	}
	if conf.EncryptKeysIds == "" {
		fmt.Print("DecryptKeyIds: ")
		fmt.Scanln(&input)
		conf.EncryptKeysIds = input
	}
	// TODO: change the logic to --prompt instead of empty Passphrase
	// or us PromptFunction from the openpgp package
	if conf.Passphrase == "" {
		fmt.Printf("Passphrase to unlock your key (%s) :", conf.DecryptKeyIds)
		pw, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}
		conf.Passphrase = string(pw)
	}
	return conf, nil

}

func main() {

	usage := `Geep password manager

Usage:
	geep read [options] <file> [<dir>] [--print]
	geep list [options] [<dir>]
	geep add [options] [--prompt]

Options:
	-p --passphrase			Prompt a passphrase
	-e --encrypt-to-key-ids	List of key ids the message should be encypted time_colon
	-d --decrypt-to-key-ids	Private Key id to decrypt the message`

	args, err := docopt.Parse(usage, nil, true, "geep cli version: 0.0.1", false)
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
			fmt.Println("fname:", fname)
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
			fmt.Printf("content:\n%s\n", string(content))
			account, err := geep.NewAccountFromString(fname, string(content))
			if err != nil {
				fmt.Println("An error occured while creating and account from file content", err)
				os.Exit(1)
			}
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

	fmt.Println(args)
	fmt.Println(conf)
}
