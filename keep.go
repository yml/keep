package keep

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jcmdev0/gpgagent"

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
				return nil, err
			}
			return []byte(passphrase), nil
		}
		return nil, fmt.Errorf("Unable to find key")
	}
}

func promptFunctionGpgAgent(conn *gpgagent.Conn) openpgp.PromptFunction {
	return func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		defer conn.Close()

		for _, key := range keys {
			cacheId := strings.ToUpper(hex.EncodeToString(key.PublicKey.Fingerprint[:]))
			fmt.Println("short key", key.PrivateKey.KeyIdShortString())

			request := gpgagent.PassphraseRequest{CacheKey: cacheId}
			passphrase, err := conn.GetPassphrase(&request)
			if err != nil {
				return nil, err
			}
			err = key.PrivateKey.Decrypt([]byte(passphrase))
			if err != nil {
				err := conn.RemoveFromCache(cacheId)
				if err != nil {
					fmt.Println("cannot remove the key from cache", err)
				}
				fmt.Println("can t decrypt", err)
				return nil, err
			}
			return []byte(passphrase), nil
		}
		return nil, fmt.Errorf("Unable to find key")
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
			return nil, err
		}
		return pw, nil
	}
	return nil, fmt.Errorf("Unable to find key")
}

func GuessPromptFunction() openpgp.PromptFunction {
	pf := promptTerminal
	conn, err := gpgagent.NewGpgAgentConn()
	if err == nil {
		pf = promptFunctionGpgAgent(conn)
	}

	// if GPGPASSPHRASE in Environ use it else request it when needed
	envs := os.Environ()
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
	SecringDir       string
	PubringDir       string
	AccountDir       string
	RecipientKeysIds string
	PromptFunction   openpgp.PromptFunction
}

func NewConfig() *Config {
	gpgkey := os.Getenv("GPGKEY")
	pubring := os.ExpandEnv(pubringDefault)
	secring := os.ExpandEnv(secringDefault)
	pwdDir := os.ExpandEnv(passwordDirDefault)

	return &Config{
		SecringDir:       secring,
		PubringDir:       pubring,
		AccountDir:       pwdDir,
		RecipientKeysIds: gpgkey,
		PromptFunction:   GuessPromptFunction(),
	}
}

func (c *Config) EncryptionRecipients() (openpgp.EntityList, error) {
	el, err := getKeyRing(c.PubringDir)
	if err != nil {
		return nil, err
	}
	el = filterEntityList(el, c.RecipientKeysIds)
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

func (c *Config) ListFileInAccount(fileSubStr string) ([]os.FileInfo, error) {
	filteredFiles := make([]os.FileInfo, 0)
	files, err := ioutil.ReadDir(c.AccountDir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if strings.Contains(f.Name(), fileSubStr) {
			filteredFiles = append(filteredFiles, f)
		}
	}
	return filteredFiles, nil
}

type Account struct {
	config   *Config
	Name     string
	Username string
	Password string
	Notes    string
}

func NewAccountFromConsole(conf *Config) (*Account, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Account Name: ")
	name, _ := reader.ReadString('\n')

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Notes: ")
	notes, _ := reader.ReadString('\n')

	fmt.Print("Enter Password (`gen` to generate a random one): ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	password := string(bytePassword)
	if password == "gen" {
		return nil, fmt.Errorf("Password generation Not Implemented")
	}
	account := Account{
		config:   conf,
		Name:     strings.TrimSpace(name),
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(password),
		Notes:    strings.TrimSpace(notes),
	}

	return &account, nil
}

func NewAccountFromFile(conf *Config, fpath string) (*Account, error) {
	clearTextReader, err := conf.DecodeFile(fpath)
	if err != nil {
		return nil, err
	}

	return NewAccountFromReader(conf, filepath.Base(fpath), clearTextReader)
}

func NewAccountFromFileContent(conf *Config, name, str string) (*Account, error) {
	a := Account{
		config: conf,
		Name:   name,
	}
	chunks := strings.Split(str, "\n")
	a.Password = chunks[0]
	a.Username = chunks[1]
	a.Notes = chunks[2]

	return &a, nil
}

func NewAccountFromReader(conf *Config, name string, r io.Reader) (*Account, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	account, err := NewAccountFromFileContent(conf, name, string(content))
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (a Account) Content() []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s", a.Password, a.Username, a.Notes))
}

func (a *Account) Encrypt() ([]byte, error) {
	el, err := a.config.EncryptionRecipients()
	if err != nil {
		return nil, err
	}

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
