package ingress

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
)

// Ingress contains the configuration of the SSH server the bastion runs
type Ingress struct {
	TCPListener     net.Listener
	SSHServerConfig *ssh.ServerConfig
}

func 

// ConfigSSHServer is used to configure the SSH server te bastion runs
func (in *Ingress) ConfigSSHServer(ak map[string]bool, privateKeyPath string) {
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
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
		PasswordCallback: nil,
		MaxAuthTries:     3,
		AuthLogCallback:  nil,
	}

	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	privateSigner, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	in.SSHServerConfig.AddHostKey(privateSigner)
}
