package auth

import (
	"errors"
	"github.com/open-bastion/open-bastion/internal/config"
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

// ParseUserPrivateKey takes the user and path of the keys directory and try to
// parse /var/lib/open-bastion/users/[user]/egress-keys/[user]
// Returns a signer if successful, an error otherwise
func ParseUserPrivateKey(user string) (ssh.Signer, error) {
	//The user should be valid as the handshake has already been validated by the bastion
	path := config.BastionConfig.UserKeysDir + user + "/egress-keys/"

	privateKeyBytes, err := ioutil.ReadFile(path + user)
	if err != nil {
		return nil, errors.New("user " + user + " failed to load private key : " + err.Error())
	}

	privateSigner, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, errors.New("user " + user + ": failed to parse private key : " + err.Error())
	}

	return privateSigner, nil
}
