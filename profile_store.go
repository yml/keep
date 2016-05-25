package keep

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Profile struct {
	Name             string
	SecringDir       string
	PubringDir       string
	AccountDir       string
	RecipientKeysIds string
}

func DefaultProfile() *Profile {
	gpgkey := os.Getenv("GPGKEY")
	pubring := os.ExpandEnv(pubringDefault)
	secring := os.ExpandEnv(secringDefault)
	accountDir := os.ExpandEnv(passwordDirDefault)

	return &Profile{
		Name:             "default",
		SecringDir:       secring,
		PubringDir:       pubring,
		AccountDir:       accountDir,
		RecipientKeysIds: gpgkey,
	}

}

type ProfileStore []Profile

func GetConfigPaths() (string, string) {
	accountDir := os.ExpandEnv(passwordDirDefault)
	return filepath.Join(filepath.Dir(accountDir), "keep.conf"), accountDir
}

func initProfileStore() (ProfileStore, error) {
	configFile, accountDir := GetConfigPaths()

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		return nil, fmt.Errorf("Do nothing because config file already exsit here : %s", configFile)
	}

	err := os.MkdirAll(accountDir, 0700)
	if err != nil {
		return nil, err
	}
	profile := DefaultProfile()
	store := make(ProfileStore, 0)
	store = append(store, *profile)
	b, err := json.MarshalIndent(store, "", "\t")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(configFile, b, 0700)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func LoadProfileStore() (ProfileStore, error) {
	configFile, _ := GetConfigPaths()

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return initProfileStore()
	}
	store := make(ProfileStore, 0)
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &store)
	if err != nil {
		return nil, err
	}
	return store, nil
}
