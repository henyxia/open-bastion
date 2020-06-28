package ingress

import (
	"context"
	"errors"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/datastore"
	"github.com/open-bastion/open-bastion/internal/egress"
	"github.com/open-bastion/open-bastion/internal/logger"
	"github.com/open-bastion/open-bastion/internal/obclient"
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
func (in *Ingress) ConfigSSHServer(ak map[string]bool, privateKeyPath string, dataStore datastore.DataStore) error {
	in.SSHServerConfig = &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			//TODO properly log that
			s, err := dataStore.GetUserStatus(c.User())

			if err != nil {
				return nil, err
			}

			if s == datastore.Inactive {
				return nil, errors.New("account deactivated")
			}

			if s != datastore.Active {
				return nil, errors.New("invalid user")
			}

			if ak[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					//TODO test that with multiple keys
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, errors.New("unknown public key for user " + c.User())
		},
		PasswordCallback: nil,
		MaxAuthTries:     3,
		AuthLogCallback:  nil,
	}

	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return errors.New("failed to load private key : " + err.Error())
	}

	privateSigner, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return errors.New("failed to parse private key : " + err.Error())
	}

	in.SSHServerConfig.AddHostKey(privateSigner)

	return nil
}

// ConfigTCPListener initialize TCPListener in the Ingress struct.
func (in *Ingress) ConfigTCPListener(address string) error {
	var err error

	in.TCPListener, err = net.Listen("tcp", address)

	if err != nil {
		return errors.New("failed to listen for connection : " + err.Error())
	}

	return nil
}

//ListenAndServe listens forever for incoming SSH connections and tries to handle them
func (in *Ingress) ListenAndServe(ctx context.Context, dataStore datastore.DataStore, config config.Config) {
	logger.Info("listening for new connections...")
	for {
		logger.Debug("waiting for a new connection...")
		client := new(obclient.Client)
		client.BackendTimeout = config.BackendTimeout

		var err error

		client.TCPConnexion, err = in.TCPListener.Accept()

		if err != nil {
			logger.WarnWithErr(err, "failed to handle the TCP connection")
			continue
		}

		go in.handleClient(ctx, client, dataStore)
	}
}

//handleClient takes a context, a client with a valid initialized connection and a DataStore, try to establish
//an SSH connection then execute the client's command (either a bastion operation or a backend connection).
func (in *Ingress) handleClient(ctx context.Context, c *obclient.Client, dataStore datastore.DataStore) {
	err := c.HandshakeSSH(in.SSHServerConfig)

	logger.UpdateClientLogCtx(ctx, c)

	if err != nil {
		logger.WarnWithCtxWithErr(ctx, err, "failed to handshake")
		return
	}

	err = c.HandleSSHConnection()

	logger.UpdateClientLogCtx(ctx, c)

	defer func() {
		if err := c.SshCommChan.Close(); err != nil {
			logger.WarnWithCtxWithErr(ctx, err, "error closing the client communication channel")
		}

		if err := c.SSHConnexion.Close(); err != nil {
			logger.WarnWithCtxWithErr(ctx, err, "error closing the SSH connection")
		}
	}()

	if err != nil {
		logger.WarnWithCtxWithErr(ctx, err, "failed to handle the TCP connection")
		return
	}

	logger.InfoWithCtx(ctx, "client connected")

	if c.BackendCommand == "bastion" {
		_ = c.RunCommand()
	} else if c.BackendCommand == "ssh" {
		egress.EstablishSSHConnection(ctx, c, dataStore)
	} else if c.BackendCommand == "telnet" {
		logger.WarnWithCtxWithErr(ctx, err, "method not implemented")
	}
}
