package client

import (
	"bytes"
	"errors"
	"net"

	"github.com/open-bastion/open-bastion/internal/egress"
	logger "github.com/open-bastion/open-bastion/internal/logger"
	"golang.org/x/crypto/ssh"
)

const sshBadRequestShell = "--- open-bastion ---\n\r" +
	"\n\r" +
	"[!] error\n\r" +
	"[!]\n\r" +
	"[!] your SSH request went through the bastion without target.\n\r" +
	"[!] to access a server simply run:\n\r" +
	"[!]\n\r" +
	"[!]     ssh BASTION_IP -- SERVER_IP\n\r" +
	"[!]\n\r" +
	"[!] this incident has been logged\n\r"

//Client represent a user and all the associated ressources
type Client struct {
	TCPConnexion net.Conn

	SSHConnexion *ssh.ServerConn
	sshChan      <-chan ssh.NewChannel
	SshCommChan  ssh.Channel

	User       string
	SSHKey     ssh.Signer
	RawCommand []byte
	Protocol   string

	BackendCommand string
	BackendUser    string
	BackendHost    string
	BackendPort    int
}

//Command ...TODO
type Command struct {
}

//HandshakeSSH ...TODO
func (client *Client) HandshakeSSH(sshConfig *ssh.ServerConfig) error {
	// func handshakeSSH(c *net.Conn, sshConfig *ssh.ServerConfig) (*ssh.ServerConn, <-chan ssh.NewChannel, error) {
	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	var reqs <-chan *ssh.Request
	var err error

	client.SSHConnexion, client.sshChan, reqs, err = ssh.NewServerConn(client.TCPConnexion, sshConfig)

	if err != nil {
		return err
	}

	client.User = client.SSHConnexion.User()

	// The incoming Request channel must be serviced.
	go func() {
		//TODO verify this is correctly cleaned up
		ssh.DiscardRequests(reqs)
	}()

	return nil
}

//HandleSSHConnection ...TODO
func (client *Client) HandleSSHConnection() error {
	var requests <-chan *ssh.Request
	// Service the incoming Channel channel.
	for newChannel := range client.sshChan {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		var err error
		client.SshCommChan, requests, err = newChannel.Accept()
		if err != nil {
			logger.WarnWithErr(err, "could not accept channel")
		}

		break
	}

	for req := range requests {
		if req.Type == "exec" {
			//The request payload is a raw byte array. Its 4 first bytes contain
			//its length so we need to remove them to correctly get the strings
			//We limit the command to 512 bytes to avoid attacks
			raw := make([]byte, 512)
			copy(raw, req.Payload[4:])
			client.RawCommand = bytes.Trim(raw, "\x00")
			break
		} else if req.Type == "shell" {
			//A shell should not be requested on the bastion
			//This is here to prevent the connexion to hang with a badly formed payload
			_, _ = client.SshCommChan.Write([]byte(sshBadRequestShell))
			return errors.New("bad request type (shell)")
		}
	}

	if len(client.RawCommand) > 0 {
		bc, err := egress.ParseBackendInfo(client.RawCommand)

		//TODO return the correct thing, I was just too lazy to change is for now
		client.BackendCommand = bc.Command
		client.BackendUser = bc.User
		client.BackendHost = bc.Host
		client.BackendPort = bc.Port

		if err != nil {
			errStr := "Unable to parse target : " + err.Error() + "\n"
			_, _ = client.SshCommChan.Write([]byte(errStr))
			return errors.New("invalid payload")
		}

		//If no user is provided for the backend, use the one connected to the bastion
		//The username is parsed during the handshake, thus it should not be a problem to
		//use it here directly
		if client.BackendUser == "" {
			client.BackendUser = client.User
		}

	} else {
		_, _ = client.SshCommChan.Write([]byte("Error : Invalid payload\n"))
		return errors.New("invalid payload")
	}

	return nil
}

//DialBackend ...TODO
func (client *Client) DialBackend() error {
	bc := egress.BackendConn{
		Command: client.BackendCommand,
		User:    client.BackendUser,
		Host:    client.BackendHost,
		Port:    client.BackendPort,
	}

	// jump to new connection
	err := egress.DialSSH(client.SshCommChan, bc, client.SSHKey)

	if err != nil {
		errStr := "Error : " + err.Error() + "\n"
		_, _ = client.SshCommChan.Write([]byte(errStr))
		return err
	}

	return nil
}

//RunCommand ...TODO
func (client *Client) RunCommand() error {

	return nil
}
