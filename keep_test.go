package keep

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func Test_NewConfig(t *testing.T) {
	c := NewConfig(nil)
	el, err := c.EntityListRecipients()
	if err != nil {
		t.Errorf("An error occured while retrieving the recipients for the ecrypted message: %v", err)
	}
	if len(el) < 1 {
		t.Errorf("No entity retrieve %d", len(el))
	}

	fmt.Println(c.SecringDir, c.PubringDir)
}

func Test_ConfigListAccountsFile(t *testing.T) {
	c := NewConfig(nil)
	c.AccountDir = "test_data/passwords"
	files, err := c.ListAccountFiles("testsuite-")
	if err != nil {
		t.Error("An error occured while listing accounts ", err)
	}
	if len(files) != 2 {
		t.Error("expected exactly 2 accounts; got :", len(files))
	}
	c.AccountDir = "does-not-exist"
	files, err = c.ListAccountFiles("")
	if !os.IsNotExist(err) {
		t.Error("Expected an os.IsNotExistError; got :", err)
	}

}

func Test_Config_decodeFile(t *testing.T) {
	c := NewConfig(nil)
	c.AccountDir = "test_data/passwords"
	_, err := c.decodeAccountFile("test_data/passwords/testsuite-signed-account")
	if err != nil {
		t.Error("An error occured while reading the file", err)
	}
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
	encryptedfile := "test_data/passwords/testsuite-signed-account"
	c := NewConfig(nil)
	el, err := c.EntityListWithSecretKey()
	if err != nil {
		t.Errorf("An error occured while decrypting the privateKey %v", err)
	}
	md, err := decodeFile(el, GuessPromptFunction(), encryptedfile)
	if err != nil {
		t.Errorf("An error occured while decoding the file : %s", err)
	}
	clearTextReader := md.UnverifiedBody

	bytess, err := ioutil.ReadAll(clearTextReader)
	if err != nil {
		t.Errorf("an error occurred while reading the clear text message %v", err)
	}
	decstr := string(bytess)

	// should be done
	log.Println("Decrypted Secret:", decstr)
}

func Test_Account_Path(t *testing.T) {
	c := NewConfig(nil)
	c.AccountDir = "/foo"
	a := &Account{
		config: c,
		Name:   "bar",
	}
	expected := "/foo/bar"
	got := a.Path()
	if expected != got {
		t.Error("An error occurred while evaluating the account path. Expected", expected, "got:", got)
	}
}

func Test_NewAccountFromFile(t *testing.T) {
	c := NewConfig(nil)
	c.AccountDir = "test_data/passwords"
	account, err := NewAccountFromFile(c, "testsuite-signed-account")
	if err != nil {
		t.Error("An error occurred while creating an account from a file", err)
	}
	if account.Username != "yml" {
		t.Error("The username is not the one expected; got :", account.Username)
	}
}

func Test_Account_EncryptFile(t *testing.T) {
	c := NewConfig(nil)
	a := Account{
		config:   c,
		Name:     "name",
		Username: "username",
		Password: "password",
		Notes:    "note",
	}
	crypt, err := a.Encrypt()
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
	got := a.Bytes()
	expected := []byte("p\nu\nn")

	if !bytes.Equal(expected, got) {
		t.Errorf("got : %s - expected : %s", got, expected)
	}
}

func Test_NewAccount(t *testing.T) {
	s := "p\nu\nn"
	a, err := newAccountFromFileContent(nil, "nameAccount", s)
	if err != nil {
		t.Errorf("An error occured while scanning an account from a string : %s", err)
	}
	if a.Password != "p" {
		t.Errorf("Not the expected password : %s", a.Password)
	}
}

var genPassCases = []int{1, 2, 3, 10}

func Test_NewPassword(t *testing.T) {
	for _, l := range genPassCases {
		passBytes, err := NewPassword(l)
		if err != nil {
			t.Errorf("An error occured while gnerating the password : %s", err)
		}
		fmt.Printf("Generated password is (length %d): %s \n", l, string(passBytes))
		if len(passBytes) != l {
			t.Errorf("passBytes lenght must be %d got : %d", l, len(passBytes))
		}
	}
}
