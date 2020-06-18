package config

import (
	"encoding/json"
	"errors"
	logger "github.com/open-bastion/open-bastion/internal/logger"
	"io/ioutil"
	"net"
	"os"
)

// Config struct contains the server configuration
type Config struct {
	PermitPasswordLogin bool   `json:"PermitPasswordLogin"`
	PermitKeyLogin      bool   `json:"PermitKeyLogin"`
	PermitRootLogin     bool   `json:"PermitRootLogin"`
	AuthorizedKeysFile  string `json:"AuthorizedKeysFile"`
	PrivateKeyFile      string `json:"PrivateKeyFile"`
	UserKeysDir         string `json:"UserKeysDir"`
	ListenPort          int    `json:"ListenPort"`
	ListenAddress       string `json:"ListenAddress"`
	Log                 Log    `json:"Log"`
	DataStoreType       string `json:"datastore"`
}

//Log contains the logger configuration
type Log struct {
	Path         string `json:"Path"`
	IsJSON       bool   `json:"IsJson"`
	Level        int    `json:"Level"`
	ReportCaller bool   `json:"ReportCaller"`
}

// ParseConfig try to open and parse the file at the specified path.
// If the path is invalid or empty, the function will try to find a config file
// at the default locations.
func ParseConfig(path string) (Config, error) {
	var c Config
	home, err := os.UserHomeDir()

	if err != nil {
		return Config{}, err
	}

	// Default values if no configuration is provided for them
	defaultConfigPaths := []string{
		"/etc/open-bastion/open-bastion-conf.json",
		home + "/.config/open-bastion/open-bastion-conf.json",
		home + "/.config/open-bastion-conf.json",
		home + "/.open-bastion/open-bastion-conf.json",
	}

	defaultPrivateKey := home + "/.ssh/id_rsa"
	defaultAuthorizedKeys := home + "/.ssh/authorized_keys"
	defaultSSHPort := 22

	defaultUserKeysDirectory := "/var/lib/open-bastion/users/"

	defaultLogDirectory := "/var/log/open-bastion/"

	configPath := ""

	if path != "" {
		_, err := os.Stat(path)

		if err != nil {
			logger.Warn("provided configuration file does not exist, using default path")
		} else {
			configPath = path
		}
	}

	if configPath == "" {
		for _, p := range defaultConfigPaths {
			_, err := os.Stat(p)

			if err == nil {
				configPath = p
				break
			}
		}
	}

	if configPath == "" {
		return Config{}, errors.New("could not open any configuration file")
	}

	f, err := os.Open(configPath)

	if err != nil {
		return Config{}, err
	}

	byteContent, err := ioutil.ReadAll(f)

	if err != nil {
		return Config{}, err
	}

	err = f.Close()

	logger.WarnWithErr(err, "could not close config file")

	if err != nil {
		return Config{}, err
	}

	if !json.Valid(byteContent) {
		return Config{}, errors.New("configuration file is not a valid JSON file")
	}

	//By default, object keys which don't have a corresponding struct field are ignored
	err = json.Unmarshal(byteContent, &c)

	if err != nil {
		return Config{}, err
	}

	if !c.PermitPasswordLogin && !c.PermitKeyLogin {
		return Config{}, errors.New("no authorized login method")
	}

	if c.ListenPort == 0 {
		c.ListenPort = defaultSSHPort
	} else if c.ListenPort > 65535 || c.ListenPort < 0 {
		return Config{}, errors.New("invalid port configuration")
	}

	if net.ParseIP(c.ListenAddress) == nil {
		return Config{}, errors.New("invalid IP address configuration")
	}

	if c.PrivateKeyFile == "" {
		c.PrivateKeyFile = defaultPrivateKey
	} else {
		_, err := os.Stat(c.PrivateKeyFile)

		if err != nil {
			return Config{}, errors.New("invalid private key file path")
		}
	}

	if c.AuthorizedKeysFile == "" {
		c.AuthorizedKeysFile = defaultAuthorizedKeys
	} else {
		_, err := os.Stat(c.AuthorizedKeysFile)

		if err != nil {
			return Config{}, errors.New("invalid authorized_keys file path")
		}
	}

	if c.UserKeysDir == "" {
		c.UserKeysDir = defaultUserKeysDirectory

		logger.Warnf("no user keys directory provided, creating and using default directory %v", defaultUserKeysDirectory)
		if _, err := os.Stat("/var/lib/open-bastion/users/"); os.IsNotExist(err) {
			err = os.MkdirAll("/var/lib/open-bastion/users/", 0660)

			if err != nil {
				return Config{}, err
			}
		}
	}

	if c.Log.Path == "" {
		c.Log.Path = defaultLogDirectory

		logger.Warnf("no logs directory provided, creating and using default directory %v", defaultLogDirectory)
		if _, err := os.Stat("/var/log/open-bastion/"); os.IsNotExist(err) {
			err = os.MkdirAll("/var/log/open-bastion/", 0660)

			if err != nil {
				return Config{}, err
			}
		}
	}

	if c.DataStoreType == "" {
		logger.Warn("no data store provided, using datastore storage")
		c.DataStoreType = "datastore"
	}

	return c, nil
}

//IsJSON returns the IsJSON field of the log config
func (c Config) IsJSON() bool {
	return c.Log.IsJSON
}

//Level returns the Level field of the log config
func (c Config) Level() int {
	return c.Log.Level
}

//ReportCaller returns the ReportCaller field of the log config
func (c Config) ReportCaller() bool {
	return c.Log.ReportCaller
}
