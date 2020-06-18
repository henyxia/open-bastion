package datastore

import (
	"errors"
	"github.com/open-bastion/open-bastion/internal/config"
	"golang.org/x/crypto/ssh"
	"os"
)

// DataStore is the interface used to access users data
type DataStore interface {
	AddUser(string, string) error
	DeleteUser(string) error
	GetUserStatus(string) (int, error)

	GetType() string

	GetRawUserEgressPrivateKey(username string) ([]byte, error)
	GetUserEgressPrivateKeySigner(username string) (ssh.Signer, error)
}

// UserInfo contains data about a user
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

// InitStore return an initialized DataStore
func InitStore(config config.Config) (DataStore, error) {
	if config.DataStoreType == "datastore" {
		var store SystemStore

		store.storeType = config.DataStoreType

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

	return nil, errors.New("not implemented")
}
