package datastore

import (
	"encoding/json"
	"errors"
	logger "github.com/open-bastion/open-bastion/internal/logger"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

const (
	InvalidUsernameErr = "invalid username"
	ReadKeyErr         = "cannot read key"
)

// SystemStore represents the datastore storage
type SystemStore struct {
	path      string
	storeType string
}

func (s SystemStore) GetType() string {
	return s.storeType
}

//AddUser add a user to the datastore and create a private key for him
func (s SystemStore) AddUser(username string, privateKeyType string) error {
	//Should we validate the username when we parse the input and considere it valid from then on
	// or should we parse it in this function?
	if !isUsernameValid(username) {
		return errors.New(InvalidUsernameErr)
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
func (s SystemStore) DeleteUser(username string) error {
	if !isUsernameValid(username) {
		return errors.New(InvalidUsernameErr)
	}

	err := os.RemoveAll(s.path + "/" + username + "/")

	if err != nil {
		return err
	}

	return nil
}

//GetUserStatus takes a username, validate it and returns the status of the user
func (s SystemStore) GetUserStatus(username string) (int, error) {
	if !isUsernameValid(username) {
		return Error, errors.New(InvalidUsernameErr)
	}

	userDir := s.path + "/" + username + "/"

	if _, err := os.Stat(userDir); !os.IsNotExist(err) {
		f, err := os.Open(userDir + "info.json")

		if err != nil {
			return Error, err
		}

		byteContent, err := ioutil.ReadAll(f)

		defer func() {
			if err := f.Close(); err != nil {
				logger.WarnfWithErr(err, "could not close info file for user %v", username)
			}
		}()

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

//GetRawUserEgressPrivateKey return the user's private key as a string
func (s SystemStore) GetRawUserEgressPrivateKey(username string) ([]byte, error) {
	if !isUsernameValid(username) {
		return nil, errors.New(InvalidUsernameErr)
	}
	//TODO validate key
	f, err := os.Open(s.path + "/" + username + "/egress-keys/" + username)

	if err != nil {
		return nil, errors.New(ReadKeyErr)
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New(ReadKeyErr)
	}

	return key, nil
}

// GetUserEgressPrivateKeySigner takes the user and path of the keys directory and try to
// parse /var/lib/open-bastion/users/[user]/egress-keys/[user]
// Returns a signer if successful, an error otherwise
func (s SystemStore) GetUserEgressPrivateKeySigner(username string) (ssh.Signer, error) {

	if !isUsernameValid(username) {
		return nil, errors.New(InvalidUsernameErr)
	}
	//TODO validate key
	f, err := os.Open(s.path + "/" + username + "/egress-keys/" + username)

	if err != nil {
		return nil, errors.New(ReadKeyErr)
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New(ReadKeyErr)
	}

	privateSigner, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, errors.New("user " + username + ": failed to parse private key : " + err.Error())
	}

	return privateSigner, nil
}

//GetRawUserEgressPublicKey return the user's private key as a string
func (s SystemStore) GetRawUserEgressPublicKey(username string) ([]byte, error) {
	if !isUsernameValid(username) {
		return nil, errors.New(InvalidUsernameErr)
	}
	//TODO validate key
	f, err := os.Open(s.path + "/" + username + "/egress-keys/" + username + ".pub")

	if err != nil {
		return nil, errors.New(ReadKeyErr)
	}

	key, err := ioutil.ReadAll(f)

	if err != nil {
		return nil, errors.New(ReadKeyErr)
	}

	return key, nil
}

func isUsernameValid(username string) bool {
	//The man recommends the following rules for a username
	//This is the regex used on debian datastore to validate a username
	//This may need to be changed
	if len(username) > 32 {
		return false
	}

	reg := regexp.MustCompile("[a-z_][a-z0-9_-]*[$]?")

	return len(reg.Find([]byte(username))) == len(username)
}
