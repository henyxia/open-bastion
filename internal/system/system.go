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

type UserInfo struct {
	Active bool `json:"active"`
	Admin  bool `json:"admin"`
}

// Represents a user status
const (
	Active = iota
	Inactive
	Invalid
	Error
)

//AddUser add a user to the system and create a private key for him
func AddUser(username string, privateKeyType string) error {
	//Should we validate the username when we parse the input and considere it valid from then on
	// or should we parse it in this function?
	if !isUsernameValid(username) {
		return errors.New("Invalid username")
	}

	us, err := GetUserStatus(username)

	if err != nil {
		return err
	}

	if us != Invalid {
		return errors.New("User " + username + " already exists")
	}

	//TODO correctly set permissions
	userKeyPath := config.BastionConfig.UserKeysDir + username + "/egress-keys/" + username

	//TODO should we output the public key on stdout when creating the user?
	if privateKeyType == "ecdsa" {
		cmd := exec.Command("ssh-keygen", "-t", "ecdsa", "-b", "521", "-f", userKeyPath)
		err = cmd.Run()

		if err != nil {
			return err
		}
	} else if privateKeyType == "rsa" {
		cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-f", userKeyPath)
		err = cmd.Run()

		if err != nil {
			return err
		}
	} else {
		return errors.New("Unknown key type")
	}

	return nil
}

//DeleteUser delete a user if it exists and its associated files
func DeleteUser(username string) error {
	//TODO

	return nil
}

//GetUserStatus takes a username, validate it and returns the status of the user
func GetUserStatus(username string) (int, error) {
	if !isUsernameValid(username) {
		return Error, errors.New("Invalid username provided")
	}

	userDir := config.BastionConfig.UserKeysDir + username + "/"

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