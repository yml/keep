package geep

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	el, err := c.EncryptionRecipients()
	if err != nil {
		t.Errorf("An error occured while retrieving the recipients for the ecrypted message: %v", err)
	}
	if len(el) < 1 {
		t.Errorf("No entity retrieve %d", len(el))
	}

	fmt.Println(c.SecringDir, c.PubringDir)
}

func Test_getKeyRing(t *testing.T) {

	path := os.ExpandEnv(pubringDefault)

	el, err := getKeyRing(path)
	if err != nil {
		t.Errorf("An error occured while opening the pubring: %v", err)
	}
	if len(el) < 1 {
		t.Errorf("No Entity in the public keyring")
	}
}

func Test_filterEntityList(t *testing.T) {

	path := os.ExpandEnv(pubringDefault)
	keyid := os.Getenv("GPGKEY")

	el, err := getKeyRing(path)
	el = filterEntityList(el, keyid)
	if err != nil {
		t.Errorf("An error occured while filtering the entity list: %v", err)
	}
	expected := 1
	got := len(el)
	if got != expected {
		t.Errorf("got: %v -- expected:%d", got, expected)
		for _, e := range el {
			if e.PrimaryKey != nil {
				t.Errorf("keyIdShortString : %v", e.PrimaryKey.KeyIdShortString())
			}
		}
	}
}

func Test_DecryptFile(t *testing.T) {
	encryptedfile := "test_data/yml_test"
	c := NewConfig()
	el, err := c.DecryptedEntityList()
	if err != nil {
		t.Errorf("An error occured while decrypting the privateKey %v", err)
	}

	// Get the encrypted file content as a []byte
	f, err := os.Open(encryptedfile)
	if err != nil {
		t.Errorf("an error occured while opening the encryptedfile %v", err)
	}
	result, err := armor.Decode(f)
	if err != nil {
		t.Errorf("An error occured while decoding the armored text: %v", err)
	}
	// Decrypt it with the contents of the private key
	md, err := openpgp.ReadMessage(result.Body, el, nil, nil)
	if err != nil {
		t.Errorf("an error occured while reading the message %v", err)
	}

	bytess, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		t.Errorf("an error occured while reading the clear text message %v", err)
	}
	decstr := string(bytess)

	// should be done
	log.Println("Decrypted Secret:", decstr)
}

func Test_EncryptFile(t *testing.T) {
	c := NewConfig()
	el, err := c.EncryptionRecipients()
	a := Account{
		Name:     "name",
		Username: "username",
		Password: "password",
		Notes:    "note",
	}
	crypt, err := a.Encrypt(el)
	if err != nil {
		t.Errorf("An error occured while encrypting the account : %v", err)
	}
	fmt.Println(string(crypt))
}

func Test_AccountString(t *testing.T) {
	a := Account{
		Username: "u",
		Password: "p",
		Notes:    "n",
	}
	got := a.Content()
	expected := []byte("p\nu\nn")

	if bytes.Equal(expected, got) {
		t.Errorf("got : %s - expected : %s", got, expected)
	}
}

func Test_NewAccount(t *testing.T) {
	s := "p\nu\nn\n"
	a, err := NewAccountFromString("nameAccount", s)
	if err != nil {
		t.Errorf("An error occured while scanning an account from a string : %s", err)
	}
	if a.Password != "p" {
		t.Errorf("Not the expected password : %s", a.Password)
	}
}
