package obclient

import (
	"errors"
	"github.com/open-bastion/open-bastion/internal/logger"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
	"strings"
)

//TODO better message?
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

var ErrInvalidPort = errors.New("invalid port option")
var ErrInvalidPayload = errors.New("invalid payload")

//Client represent a user and all the associated resources.
type Client struct {
	TCPConnexion net.Conn

	SSHConnexion *ssh.ServerConn
	sshChan      <-chan ssh.NewChannel
	SshCommChan  ssh.Channel

	User       string
	SSHKey     ssh.Signer
	RawCommand []byte

	BackendCommand string
	BackendUser    string
	BackendHost    string
	BackendPort    int
}

// BackendConn contains the information to establish a connection to a backend.
type BackendConn struct {
	Command string
	User    string
	Host    string
	Port    int
}

//HandshakeSSH takes an initialized SSH configuration and tries to establish a SSH connection for the client.
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

//HandleSSHConnection handles the incoming connection. It services the incoming channel by discarding unwanted types
//then parse the request and validate its type (must be "exec"). It then parses the backend information and updates
//the client struct accordingly.
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
			if len(req.Payload) > 512 {
				_, _ = client.SshCommChan.Write([]byte("Your backend command is longer than allowed (> 512B)"))
				return ErrInvalidPayload
			}

			client.RawCommand = req.Payload[4:]
			break
		} else if req.Type == "shell" {
			//A shell should not be requested on the bastion
			//This is here to prevent the connexion to hang with a badly formed payload
			_, _ = client.SshCommChan.Write([]byte(sshBadRequestShell))
			return errors.New("bad request type (shell)")
		}
	}

	if len(client.RawCommand) > 0 {
		bc, err := ParseBackendInfo(client.RawCommand)

		if err != nil {
			errStr := "Unable to parse target : " + err.Error() + "\n"
			_, _ = client.SshCommChan.Write([]byte(errStr))
			return ErrInvalidPayload
		}

		//TODO return the correct thing, I was just too lazy to change is for now
		client.BackendCommand = bc.Command
		client.BackendUser = bc.User
		client.BackendHost = bc.Host
		client.BackendPort = bc.Port

		//If no user is provided for the backend, use the one connected to the bastion
		//The username is parsed during the handshake, thus it should not be a problem to
		//use it here directly
		if client.BackendUser == "" {
			client.BackendUser = client.User
		}

	} else {
		_, _ = client.SshCommChan.Write([]byte("Error : Invalid payload\n"))
		return ErrInvalidPayload
	}

	return nil
}

// ParseBackendInfo takes a string containing our payload command and returns
// a BackendConn struct with the required infos to call DialSSH.
func ParseBackendInfo(rawPayload []byte) (bc BackendConn, err error) {
	//Size of the payload has already been checked
	payload := string(rawPayload)

	//Remove leading and trailing whitespaces
	payload = strings.TrimSpace(payload)

	command := strings.Split(payload, " ")

	//The raw payload should at least contain a command and an argument (host...)
	if command == nil || len(command) < 2 {
		return bc, ErrInvalidPayload
	}

	//The first string should always be the command
	//This is not nil as the length is already verified
	c := command[0]

	if c == "ssh" {
		bc.Command = "ssh"
	} else if c == "telnet" {
		bc.Command = "telnet"
	} else if c == "bastion" {
		bc.Command = "bastion"
	} else {
		return BackendConn{}, errors.New("command not found")
	}

	//Parse command arguments
	for i := 0; i < len(command); i++ {
		if command[i] == "" {
			continue
		} else if command[i] == "-p" {
			//Parse port option
			if i+1 < len(command) {
				port, err := strconv.Atoi(command[i+1])

				if err != nil {
					return bc, ErrInvalidPort
				}

				if port > 65535 || port < 0 {
					return bc, ErrInvalidPort
				}

				bc.Port = port
				//Don't go over the next parameter as it already as been read
				i = i + 1
			} else {
				return bc, ErrInvalidPort
			}
		} else {
			//Parse user and host
			arr := strings.Split(command[i], `@`)

			if len(arr) == 0 {
				return bc, errors.New("could not parse destination")
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
				return bc, errors.New("could not parse destination")
			}
		}
	}

	//Backend connection default to port 22
	if bc.Command == "ssh" && bc.Port == 0 {
		bc.Port = 22
	}

	if bc.Host == "" {
		return bc, errors.New("could not parse backend parameters")
	}

	return bc, nil
}

//RunCommand not implemented.
func (client *Client) RunCommand() error {

	return errors.New("not implemented")
}

//GetUser implements the ClientInfoGetter. It returns the client's User.
func (client Client) GetUser() string {
	return client.User
}

//GetIp implements the ClientInfoGetter. It returns the client's remote IP address.
func (client Client) GetIp() string {
	return client.TCPConnexion.RemoteAddr().String()
}

//GetPublicKeyFingerprint implements the ClientInfoGetter. It returns the client's public key fingerprint or
//an empty string if it is not initialized.
func (client Client) GetPublicKeyFingerprint() string {
	if client.SSHKey != nil && client.SSHKey.PublicKey() != nil {
		return ssh.FingerprintSHA256(client.SSHKey.PublicKey())
	}

	return ""
}

//GetCommand implements the ClientInfoGetter. It returns the client's raw backend command.
func (client Client) GetCommand() string {
	return string(client.RawCommand)
}

//GetBackendCommand implements the ClientInfoGetter. It returns a valid backend command.
func (client Client) GetBackendCommand() string {
	return client.BackendCommand
}

//GetBackendUser implements the ClientInfoGetter. It returns the user passed to the backend command.
func (client Client) GetBackendUser() string {
	return client.BackendUser
}

//GetBackendHost implements the ClientInfoGetter. It returns the backend host.
func (client Client) GetBackendHost() string {
	return client.BackendHost
}

//GetBackendPort implements the ClientInfoGetter. It returns the backend port.
func (client Client) GetBackendPort() int {
	return client.BackendPort
}
