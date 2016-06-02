package keep

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
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
	validChars = "abcdefijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ0123456789~-_=+(){}@#&$€"
)

// NewPassword return a randomly generated password of the requested length
func NewPassword(length int) ([]byte, error) {
	password := make([]byte, length)
	l := int64(len(validChars) - 1)
	for i := 0; i < length; i++ {
		randN, err := rand.Int(rand.Reader, big.NewInt(l))
		if err != nil {
			return nil, err
		}
		password[i] = validChars[randN.Int64()]
	}
	return password, nil
}

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

func decodeFile(el openpgp.EntityList, pf openpgp.PromptFunction, fpath string) (*openpgp.MessageDetails, error) {
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
	return openpgp.ReadMessage(result.Body, el, pf, nil)
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
			cacheID := strings.ToUpper(hex.EncodeToString(key.PublicKey.Fingerprint[:]))

			fmt.Println("Private key short ID :", key.Entity.PrivateKey.KeyIdShortString())

			request := gpgagent.PassphraseRequest{CacheKey: cacheID}
			passphrase, err := conn.GetPassphrase(&request)
			if err != nil {
				return nil, err
			}
			err = key.PrivateKey.Decrypt([]byte(passphrase))
			if err != nil {
				err := conn.RemoveFromCache(cacheID)
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

// GuessPromptFunction is a function that returns an openpgp.PromptFunction well suited for the context.
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
			fmt.Println("Overriding PromptFunction to use Environ")
			pf = promptFromString(env[1])
			break
		}
	}
	return pf
}

// Config represents the configuration required to work with GPG.
type Config struct {
	SecringDir      string
	PubringDir      string
	AccountDir      string
	RecipientKeyIds string
	SignerKeyID     string
	PromptFunction  openpgp.PromptFunction
}

// NewConfig returns an initialized Config with the information copied from a Profile. If nil Profile is passed we build one from DefaultProfile.
func NewConfig(p *Profile) *Config {
	if p == nil {
		p = DefaultProfile()
	}
	return &Config{
		SecringDir:      p.SecringDir,
		PubringDir:      p.PubringDir,
		AccountDir:      p.AccountDir,
		RecipientKeyIds: p.RecipientKeyIds,
		SignerKeyID:     p.SignerKeyID,
		PromptFunction:  GuessPromptFunction(),
	}
}

// EntityListRecipients returns the openpgp.EntityList corresponding to the RecipientKeyIds from the Config.
func (c *Config) EntityListRecipients() (openpgp.EntityList, error) {
	el, err := getKeyRing(c.PubringDir)
	if err != nil {
		return nil, err
	}
	el = filterEntityList(el, c.RecipientKeyIds)
	return el, nil
}

// EntityListWithSecretKey returns the openpgp.EntityList contains in Secring.
func (c *Config) EntityListWithSecretKey() (openpgp.EntityList, error) {
	el, err := getKeyRing(c.SecringDir)
	if err != nil {
		return nil, err
	}
	return el, nil
}

// EntitySigner returns an Entity with a decrypted Private Key.
func (c *Config) EntitySigner() (*openpgp.Entity, error) {
	el, err := getKeyRing(c.SecringDir)
	if err != nil {
		return nil, err
	}
	el = filterEntityList(el, c.SignerKeyID)

	if len(el) != 1 {
		return nil, fmt.Errorf("Exactly one SignerKeyID must be given, received : %d", len(el))
	}

	// Decrypt the private key
	prompt := GuessPromptFunction()
	passphrase, err := prompt(el.DecryptionKeys(), false)
	if err != nil {
		return nil, err
	}
	signer := el[0]
	err = signer.PrivateKey.Decrypt(passphrase)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// decodeFile returns an io.Reader from which the content of the message can be read in clear text.
func (c *Config) decodeFile(fpath string) (*openpgp.MessageDetails, error) {
	el, err := c.EntityListWithSecretKey()
	if err != nil {
		return nil, err
	}
	return decodeFile(el, c.PromptFunction, fpath)
}

// ListAccountFiles returns the list of Files stored in the AccountDir.
// The list is filtered in a case in sensitive way.
func (c *Config) ListAccountFiles(fileSubStr string) ([]os.FileInfo, error) {
	var filteredFiles []os.FileInfo
	files, err := ioutil.ReadDir(c.AccountDir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if strings.Contains(strings.ToLower(f.Name()), strings.ToLower(fileSubStr)) {
			filteredFiles = append(filteredFiles, f)
		}
	}
	return filteredFiles, nil
}

// Account represents an Account
type Account struct {
	config   *Config
	Name     string
	Username string
	Password string
	Notes    string

	// The following fields are valued when the account is read.
	IsSigned bool
	SignedBy *openpgp.Key // the key of the signer, if available
}

// NewAccountFromConsole returns an Account built with the elements collected by interacting with the user.
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

	// Making sure that we jump a line in the console after reading the Password
	fmt.Printf("\n")

	if bytes.Equal(bytePassword, []byte("gen")) {
		bytePassword, err = NewPassword(10)
		if err != nil {
			return nil, err
		}
	}
	password := string(bytePassword)

	account := Account{
		config:   conf,
		Name:     strings.TrimSpace(name),
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(password),
		Notes:    strings.TrimSpace(notes),
	}

	return &account, nil
}

// NewAccountFromFile returns an Account as described by a file in the accountDir.
func NewAccountFromFile(conf *Config, fpath string) (*Account, error) {
	md, err := conf.decodeFile(fpath)
	if err != nil {
		return nil, err
	} else if md.IsSigned && md.SignatureError != nil {
		return nil, fmt.Errorf("A signature error has been detected in this accoun : %v", err)
	}

	clearTextReader := md.UnverifiedBody
	account, err := NewAccountFromReader(conf, filepath.Base(fpath), clearTextReader)

	if md.IsSigned {
		account.IsSigned = true
		account.SignedBy = md.SignedBy
	}

	return account, err
}

func newAccountFromFileContent(conf *Config, name, str string) (*Account, error) {
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

// NewAccountFromReader returns an account with the provided element.
// The reader is expected to returns bytes int the appropriate format :
//   * []byte(password\nusername\nnotes)
func NewAccountFromReader(conf *Config, name string, r io.Reader) (*Account, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	account, err := newAccountFromFileContent(conf, name, string(content))
	if err != nil {
		return nil, err
	}
	return account, nil
}

// Bytes returns a slice of byte representing the account.
func (a Account) Bytes() []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s", a.Password, a.Username, a.Notes))
}

// Encrypt returns the encrypted byte slice for an account.
func (a *Account) Encrypt() ([]byte, error) {
	el, err := a.config.EntityListRecipients()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	aw, err := armor.Encode(
		buf,
		"PGP MESSAGE",
		map[string]string{"Version": "OpenPGP"},
	)
	var signer *openpgp.Entity
	if a.config.SignerKeyID != "" {
		signer, err = a.config.EntitySigner()
		if err != nil {
			return nil, err
		}
	}

	w, err := openpgp.Encrypt(aw, el, signer, nil, nil)
	if err != nil {
		return nil, err
	}

	_, err = w.Write(a.Bytes())
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
