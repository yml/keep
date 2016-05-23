package geep

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	secringDefault     = "$HOME/.gnupg/secring.gpg"
	pubringDefault     = "$HOME/.gnupg/pubring.gpg"
	passwordDirDefault = "$HOME/.kip/passwords"
)

func getKeyRing(keyringPath string) (el openpgp.EntityList, err error) {
	// Read in public key
	keyringFileBuffer, err := os.Open(keyringPath)
	if err != nil {
		return nil, err
	}
	defer keyringFileBuffer.Close()

	el, err = openpgp.ReadKeyRing(keyringFileBuffer)
	if err != nil {
		return nil, err
	}
	return el, nil
}

func filterEntityList(el openpgp.EntityList, recipients string) openpgp.EntityList {
	rs := strings.Split(recipients, " ")
	fel := make([]*openpgp.Entity, 0, len(rs))
	for _, r := range rs {
		for _, e := range el {
			if r == e.PrimaryKey.KeyIdShortString() {
				fel = append(fel, e)
			}
		}
	}
	return fel
}

func decodeFile(el openpgp.EntityList, pf openpgp.PromptFunction, fpath string) (io.Reader, error) {
	// Get the encrypted file content as a []byte
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	result, err := armor.Decode(f)
	if err != nil {
		return nil, err
	}

	// Decrypt it with the contents of the private key
	md, err := openpgp.ReadMessage(result.Body, el, pf, nil)
	if err != nil {
		return nil, err
	}
	return md.UnverifiedBody, nil

}

func promptFromString(passphrase string) openpgp.PromptFunction {
	return func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		for _, k := range keys {
			ID := k.PrivateKey.KeyIdShortString()
			fmt.Printf("Passphrase to unlock your key (%s) : ", ID)
			err := k.PrivateKey.Decrypt([]byte(passphrase))
			if err != nil {
				fmt.Println("\nAn error occurred while decrypting the key", err)
				continue
			}
			break

		}
		return nil, nil
	}
}

func promptTerminal(keys []openpgp.Key, symmetric bool) ([]byte, error) {
	for _, k := range keys {
		ID := k.PrivateKey.KeyIdShortString()
		fmt.Printf("Passphrase to unlock your key (%s) : ", ID)
		pw, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}
		err = k.PrivateKey.Decrypt(pw)
		if err != nil {
			fmt.Println("\nAn error occurred while decrypting the key", err)
			continue
		}
		break

	}
	return nil, nil
}

func GuessPromptFunction() openpgp.PromptFunction {
	// if GPGPASSPHRASE in Environ use it else request it when needed
	envs := os.Environ()
	pf := promptTerminal
	for _, val := range envs {
		env := strings.Split(val, "=")
		if len(env) == 2 && env[0] == "GPGPASSPHRASE" {
			pf = promptFromString(env[1])
			break
		}
	}
	return pf
}

type Config struct {
	SecringDir     string
	PubringDir     string
	PasswordDir    string
	EncryptKeysIds string
	PromptFunction openpgp.PromptFunction
}

func NewConfig() *Config {
	gpgkey := os.Getenv("GPGKEY")
	pubring := os.ExpandEnv(pubringDefault)
	secring := os.ExpandEnv(secringDefault)
	pwdDir := os.ExpandEnv(passwordDirDefault)

	return &Config{
		SecringDir:     secring,
		PubringDir:     pubring,
		PasswordDir:    pwdDir,
		EncryptKeysIds: gpgkey,
		PromptFunction: GuessPromptFunction(),
	}
}

func (c *Config) EncryptionRecipients() (openpgp.EntityList, error) {
	el, err := getKeyRing(c.PubringDir)
	if err != nil {
		return nil, err
	}
	el = filterEntityList(el, c.EncryptKeysIds)
	return el, nil
}

func (c *Config) DecryptedEntityList() (openpgp.EntityList, error) {
	el, err := getKeyRing(c.SecringDir)
	if err != nil {
		return nil, err
	}
	return el, nil

}

func (c *Config) DecodeFile(fpath string) (io.Reader, error) {
	el, err := c.DecryptedEntityList()
	if err != nil {
		return nil, err
	}
	return decodeFile(el, c.PromptFunction, fpath)
}

type Account struct {
	Name     string
	Username string
	Password string
	Notes    string
}

func NewAccountFromString(name, str string) (*Account, error) {
	a := Account{Name: name}
	_, err := fmt.Sscanf(
		str,
		"%s\n%s\n%s", &a.Password, &a.Username, &a.Notes,
	)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (a Account) Content() []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s\n", a.Password, a.Username, a.Notes))
}

func (a *Account) Encrypt(el openpgp.EntityList) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	aw, err := armor.Encode(
		buf,
		"PGP MESSAGE",
		map[string]string{"Version": "OpenPGP"},
	)
	w, err := openpgp.Encrypt(aw, el, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	_, err = w.Write(a.Content())
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	err = aw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
