package egress

import (
	"errors"
	"strings"
	"github.com/open-bastion/open-bastion/internal/egress/ssh"
	"github.com/open-bastion/open-bastion/internal/egress/backend"
)

// ParseBackendInfo check user input and send it to the proper egress module
func ParseBackendInfo(payload string) (bc backend.Conn, err error) {
	// check payload size
	if len(payload) == 0 {
		return bc, errors.New("empty payload")
	}

	if len(payload) > 1024 {
		return bc, errors.New("payload too long")
	}

	// remove leading and trailing whitespaces
	payload = strings.TrimSpace(payload)

	command := strings.Split(payload, " ")

	// check payload content
	if command == nil {
		return bc, errors.New("invalid payload")
	}

	if len(command) < 2 {
		return bc, errors.New("payload too short")
	}

	// redirect to the chosen egress
	if command[0] == "ssh" {
		return ssh.ParseBackendInfo(command)
	}

	return bc, errors.New("invalid egress")
}
