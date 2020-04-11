package system

import (
	"encoding/json"
	"errors"
	"github.com/open-bastion/open-bastion/internal/config"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// SystemStore represents the system storage
type SystemStore struct {
	path string
}

// InitSystemStore return an initialized DataStore
func InitSystemStore(config config.Config) (SystemStore, error) {
	var store SystemStore

	_, err := os.Stat(config.UserKeysDir)

	if os.IsNotExist(err) {
		err := os.MkdirAll(config.UserKeysDir, 0600)

		if err != nil {
			return SystemStore{}, errors.New("error, cannot create data store")
		}
	}

	store.path = config.UserKeysDir

	return store, nil
}

//AddUser add a user to the system and create a private key for him
func (s SystemStore) AddUser(username string, privateKeyType string) error {
	//Should we validate the username when we parse the input and considere it valid from then on
	// or should we parse it in this function?
	if !isUsernameValid(username) {
		return errors.New("Invalid username")
	}

	us, err := s.GetUserStatus(username)

	if err != nil {
		return err
	}

	if us != Invalid {
		return errors.New("User " + username + " already exists")
	}

	userKeyPath := s.path + username + "/egress-keys/" + username

	//TODO should we output the public key on stdout when creating the user?
	if privateKeyType == "ecdsa" {
		cmd := exec.Command("ssh-keygen", "-t", "ecdsa", "-b", "521", "-f", userKeyPath)
		err = cmd.Run()
	} else if privateKeyType == "rsa" {
		cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-f", userKeyPath)
		err = cmd.Run()
	} else {
		return errors.New("Unknown key type")
	}

	if err != nil {
		return err
	}

	err = os.Chmod(userKeyPath, 0600)

	if err != nil {
		return err
	}

	return nil
}

//DeleteUser delete a user if it exists and its associated files
func (s SystemStore) DeleteUser(username string) error {
	// Sanitize input
	// TODO better sanitizing, maybe move it in the command parsing
	if strings.Contains(username, "/.") {
		return errors.New("username contains invalid character")
	}

	_, err := os.Stat(s.path + username + "/")

	if err != nil {
		return err
	}

	os.RemoveAll(s.path + username + "/")

	return nil
}

//GetUserStatus takes a username, validate it and returns the status of the user
func (s SystemStore) GetUserStatus(username string) (int, error) {
	if !isUsernameValid(username) {
		return Error, errors.New("Invalid username provided")
	}

	userDir := s.path + username + "/"

	if _, err := os.Stat(userDir); !os.IsNotExist(err) {
		f, err := os.Open(userDir + "info.json")

		if err != nil {
			return Error, err
		}

		byteContent, err := ioutil.ReadAll(f)

		f.Close()

		if err != nil {
			return Error, err
		}

		if !json.Valid(byteContent) {
			return Error, errors.New("The configuration file is not a valid JSON file")
		}

		var ui UserInfo
		err = json.Unmarshal([]byte(byteContent), &ui)

		if err != nil {
			return Error, err
		}

		if ui.Active {
			return Active, nil
		}

		return Inactive, nil
	}

	return Invalid, nil
}

//GetUserEgressPrivateKey return the user's private key as a string
func (s SystemStore) GetUserEgressPrivateKey(username string) ([]byte, error) {
	status, err := s.GetUserStatus(username)

	if err != nil {
		return nil, errors.New("error getting user status")
	}

	if status != Active {
		return nil, errors.New("error, user not active")
	}

	f, err := os.Open(s.path + username + "/egress-keys/" + username)

	if err != nil {
		return nil, errors.New("error reading key")
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New("error reading key")
	}

	return key, nil
}

//GetUserEgressPublicKey return the user's private key as a string
func (s SystemStore) GetUserEgressPublicKey(username string) ([]byte, error) {
	status, err := s.GetUserStatus(username)

	if err != nil {
		return nil, errors.New("error getting user status")
	}

	if status != Active {
		return nil, errors.New("error, user not active")
	}

	f, err := os.Open(s.path + username + "/egress-keys/" + username + ".pub")

	if err != nil {
		return nil, errors.New("error reading key")
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New("error reading key")
	}

	return key, nil
}

func isUsernameValid(username string) bool {
	//The man recommends the following rules for a username
	//This is the regex used on debian system to validate a username
	//This may need to be changed
	if len(username) > 32 {
		return false
	}

	reg := regexp.MustCompile("[a-z_][a-z0-9_-]*[$]?")

	if reg.Match([]byte(username)) == false {
		return false
	}

	return true
}
