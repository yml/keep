package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/docopt/docopt-go"
	"github.com/yml/geep"
)

var input string

func main() {
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
			fmt.Println("An error occured while reading the password", err)
			os.Exit(1)
		}
		conf.Passphrase = string(pw)
	}

	usage := `Usage: geep [options] <file> [<dir>] [--print]
geep ls [options] [<dir>]
geep add [options] [--prompt]

Options:
  -p --passphrase			Prompt a passphrase
  -e --encrypt-to-key-ids	List of key ids the message should be encypted time_colon
  -d --decrypt-to-key-ids	Private Key id to decrypt the message`

	args, err := docopt.Parse(usage, nil, true, "geep cli version: 0.0.1", false)
	if err != nil {
		log.Fatal("err: ", err)
	}
	fmt.Println(args)
}
