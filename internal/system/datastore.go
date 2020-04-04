package system

import ()

// DataStore is the interface used to access users data
type DataStore interface {
	AddUser(string, string) error
	DeleteUser(string) error
	GetUserStatus(string) (int, error)
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
