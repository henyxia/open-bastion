package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
<<<<<<< HEAD
	PermitPasswordLogin bool `json:"PermitPasswordLogin"`
=======
	PermitPasswordLogin bool   `json:"PermitPasswordLogin"`
	PermitKeyLogin      bool   `json:"PermitKeyLogin"`
	PermitRootLogin     bool   `json:"PermitRootLogin"`
	AuthorizedKeysFile  string `json:"AuthorizedKeysFile"`
	ListenPort          int    `json:"ListenPort"`
	ListenAddress       string `json:"ListenAddress"`
>>>>>>> 97a0549... Crated first proto with project structure.
}

func (c *Config) ParseConfig(path string) {
	f, err := os.Open(path)

	if err != nil {
		panic(err)
	}

	byteContent, err := ioutil.ReadAll(f)

	if err != nil {
		panic(err)
	}

	if !json.Valid(byteContent) {
		panic("The configuration file is not a valid JSON file.")
	}

	//By default, object keys which don't have a corresponding struct field are ignored
	json.Unmarshal([]byte(byteContent), &c)

	f.Close()
}
