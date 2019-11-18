package auth

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
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

	byteContent, err := ioutil.ReadAll(f)
	f.Close()

	if err != nil {
		return err
	}

	for len(byteContent) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(byteContent)
		if err != nil {
			return err
		}

		ak.AuthorizedKeys[string(pubKey.Marshal())] = true
		byteContent = rest
	}
	return nil
}
