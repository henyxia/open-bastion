package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"log"
	"github.com/pkg/errors"
)

type Config struct {
	PermitPasswordLogin bool   `json:"PermitPasswordLogin"`
	PermitKeyLogin      bool   `json:"PermitKeyLogin"`
	PermitRootLogin     bool   `json:"PermitRootLogin"`
	AuthorizedKeysFile  string `json:"AuthorizedKeysFile"`
	ListenPort          int    `json:"ListenPort"`
	ListenAddress       string `json:"ListenAddress"`
}

func GetConfigurationPath() (string, error) {
	// get home dir
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Unable to retrieve user home")
		log.Fatalln(err)
		return "", errors.New("unable to retrieve user home")
	}

	// set paths
	var paths [3]string
	paths[0] = home+"/.config/open-bastion/config.json"
	paths[1] = home+"/.open-bastion/config.json"
	paths[2] = "/etc/open-bastion/config.json"

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", errors.New("no configuration found")
}

func (c *Config) ParseConfig(path string) (error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	byteContent, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	if !json.Valid(byteContent) {
		return errors.New("The configuration file is not a valid JSON file.")
	}

	//By default, object keys which don't have a corresponding struct field are ignored
	json.Unmarshal([]byte(byteContent), &c)

	f.Close()

	return nil
}
