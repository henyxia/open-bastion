package auth

import (
	"bufio"
	logger "github.com/open-bastion/open-bastion/internal/logger"
	"golang.org/x/crypto/ssh"
	"os"
)

// Auth contains the information to authenticate the clients
type Auth struct {
	AuthorizedKeys map[string]bool
}

// ReadAuthorizedKeysFile reads the specified file and populate a map of the parsed keys
func (ak *Auth) ReadAuthorizedKeysFile(path string) (err error) {
	ak.AuthorizedKeys = make(map[string]bool)
	f, err := os.Open(path)

	if err != nil {
		return err
	}

	line := 1
	//TODO check if there is a line length limitation for the scanner
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey(scanner.Bytes())
		if err != nil {
			logger.WarnfWithErr(err, "error reading key line %v", line)
		} else {
			ak.AuthorizedKeys[string(pubKey.Marshal())] = true
		}

		line++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
