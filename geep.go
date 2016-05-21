package geep

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
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

func decryptPrivateKey(entity *openpgp.Entity, passphrase string) error {
	// Get the passphrase and read the private key.
	// Have not touched the encrypted string yet
	passphrasebyte := []byte(passphrase)
	log.Println("Decrypting private key using passphrase: ")
	err := entity.PrivateKey.Decrypt(passphrasebyte)
	if err != nil {
		return nil
	}
	// TODO: I am not sure the loop below is required
	for _, subkey := range entity.Subkeys {
		err := subkey.PrivateKey.Decrypt(passphrasebyte)
		if err != nil {
			return err
		}
	}
	return nil
}

type Config struct {
	Passphrase     string
	SecringDir     string
	PubringDir     string
	PasswordDir    string
	EncryptKeysIds string
	DecryptKeyIds  string
}

func NewConfig() *Config {
	passphrase := os.Getenv("PASSPHRASE")
	gpgkey := os.Getenv("GPGKEY")
	pubring := os.ExpandEnv(pubringDefault)
	secring := os.ExpandEnv(secringDefault)
	pwdDir := os.ExpandEnv(passwordDirDefault)
	return &Config{
		Passphrase:     passphrase,
		SecringDir:     secring,
		PubringDir:     pubring,
		PasswordDir:    pwdDir,
		EncryptKeysIds: gpgkey,
		DecryptKeyIds:  gpgkey,
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
	el = filterEntityList(el, c.DecryptKeyIds)
	fmt.Println("el length", len(el))
	err = decryptPrivateKey(el[0], c.Passphrase)
	if err != nil {
		return nil, err
	}
	return el, nil

}

type Account struct {
	Name     string
	Username string
	Password string
	Notes    string
}

func NewAccountFromString(name, str string) (*Account, error) {
	a := Account{Name: name}
	n, err := fmt.Sscanf(
		str,
		"%s\n%s\n%s", &a.Password, &a.Username, &a.Notes,
	)
	if err != nil {
		return nil, err
	}
	fmt.Println("n :", n)
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
