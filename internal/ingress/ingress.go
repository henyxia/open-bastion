package ingress

import (
	"errors"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
)

// Ingress contains the configuration of the SSH server the bastion runs
type Ingress struct {
	TCPListener     net.Listener
	SSHServerConfig *ssh.ServerConfig
}

// ConfigSSHServer is used to configure the SSH server the bastion runs
func (in *Ingress) ConfigSSHServer(ak map[string]bool, privateKeyPath string) error {
	in.SSHServerConfig = &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if ak[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, errors.New("Unknown public key for " + c.User())
		},
		PasswordCallback: nil,
		MaxAuthTries:     3,
		AuthLogCallback:  nil,
	}

	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return errors.New("Failed to load private key : " + err.Error())
	}

	privateSigner, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return errors.New("Failed to parse private key : " + err.Error())
	}

	in.SSHServerConfig.AddHostKey(privateSigner)

	return nil
}

// ConfigTCPListener initializee TCPListener in the Ingress struct.
func (in *Ingress) ConfigTCPListener(address string) error {
	var err error

	in.TCPListener, err = net.Listen("tcp", address)

	if err != nil {
		return errors.New("Failed to listen for connection : " + err.Error())
	}

	return nil
}
