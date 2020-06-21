package ssh

import (
	"errors"
	"strconv"
	"strings"

	"github.com/open-bastion/open-bastion/internal/egress/backend"
)

// ParseBackendInfo takes a string containing our payload command and returns
// a BackendConn struct with the required infos to call DialSSH
func ParseBackendInfo(commands []string) (bc backend.Conn, err error) {
	for i := 0; i < len(commands); i++ {
		if commands[i] == "" {
			continue
		} else if commands[i] == "-p" {
			if i+1 < len(commands) {
				port, err := strconv.Atoi(commands[i+1])

				if err != nil {
					return bc, errors.New("Invalid port option")
				}

				if port > 65535 || port < 0 {
					return bc, errors.New("Invalid port option")
				}

				bc.Port = port
				//Don't go over the next parameter as it already as been read
				i = i + 1
			} else {
				return bc, errors.New("Invalid port option")
			}
		} else {
			arr := strings.Split(commands[i], `@`)

			if len(arr) == 0 {
				return bc, errors.New("Could not parse destination")
			}

			if len(arr) == 1 {
				bc.Host = arr[0]
				continue
			}

			if len(arr) == 2 {
				bc.User = arr[0]
				bc.Host = arr[1]
				continue
			}

			if len(arr) > 2 {
				return bc, errors.New("Could not parse destination")
			}
		}
	}

	if bc.Port == 0 {
		bc.Port = 22
	}

	if bc.Host == "" {
		return bc, errors.New("Could not parse backend parameters")
	}

	return bc, nil
}
