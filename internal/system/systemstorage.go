package system

import (
	"encoding/json"
	"errors"
	"github.com/open-bastion/open-bastion/internal/config"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

// Store represents the system storage
type Store struct {
	path string
}

// InitStore return an initialized DataStore
func InitStore(config config.Config) (Store, error) {
	var store Store

	_, err := os.Stat(config.UserKeysDir)

	if os.IsNotExist(err) {
		err := os.MkdirAll(config.UserKeysDir, 0600)

		if err != nil {
			return SystemStore{}, errors.New("cannot create data store")
		}
	}

	store.path = config.UserKeysDir

	return store, nil
}

//AddUser add a user to the system and create a private key for him
func (s Store) AddUser(username string, privateKeyType string) error {
	//Should we validate the username when we parse the input and considere it valid from then on
	// or should we parse it in this function?
	if !isUsernameValid(username) {
		return errors.New("invalid username")
	}

	us, err := s.GetUserStatus(username)

	if err != nil {
		return err
	}

	if us != Invalid {
		return errors.New("user " + username + " already exists")
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
		return errors.New("unknown key type")
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
func (s Store) DeleteUser(username string) error {
	if !isUsernameValid(username) {
		return errors.New("invalid username")
	}

	err := os.RemoveAll(s.path + "/" + username + "/")

	if err != nil {
		return err
	}

	return nil
}

//GetUserStatus takes a username, validate it and returns the status of the user
func (s Store) GetUserStatus(username string) (int, error) {
	if !isUsernameValid(username) {
		return Error, errors.New("invalid username")
	}

	userDir := s.path + "/" + username + "/"

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
			return Error, errors.New("configuration file is not a valid JSON file")
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

	return Error, errors.New("user does not exist")
}

//GetUserEgressPrivateKey return the user's private key as a string
func (s Store) GetUserEgressPrivateKey(username string) ([]byte, error) {
	if !isUsernameValid(username) {
		return nil, errors.New("invalid username")
	}
	//TODO validate key
	f, err := os.Open(s.path + "/" + username + "/egress-keys/" + username)

	if err != nil {
		return nil, errors.New("cannot read key")
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New("read key")
	}

	return key, nil
}

//GetUserEgressPublicKey return the user's private key as a string
func (s Store) GetUserEgressPublicKey(username string) ([]byte, error) {
	if !isUsernameValid(username) {
		return nil, errors.New("invalid username")
	}
	//TODO validate key
	f, err := os.Open(s.path + "/" + username + "/egress-keys/" + username + ".pub")

	if err != nil {
		return nil, errors.New("cannot read key")
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New("cannot read key")
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

	if len(reg.Find([]byte(username))) != len(username) {
		return false
	}

	return true
}
