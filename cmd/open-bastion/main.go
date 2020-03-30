package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/open-bastion/open-bastion/internal/auth"
	"github.com/open-bastion/open-bastion/internal/config"
	"github.com/open-bastion/open-bastion/internal/egress"
	"github.com/open-bastion/open-bastion/internal/ingress"
	"github.com/open-bastion/open-bastion/internal/logs"
	"github.com/open-bastion/open-bastion/internal/system"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"strconv"
)

//Client represent a user and all the associated ressources
type Client struct {
	TCPConnexion net.Conn

	SSHConnexion *ssh.ServerConn
	sshChan      <-chan ssh.NewChannel
	sshCommChan  ssh.Channel

	User       string
	SSHKey     ssh.Signer
	RawCommand string
	Protocol   string

	BackendCommand string
	BackendUser    string
	BackendHost    string
	BackendPort    int
}

func main() {
	var configPath = flag.String("config-file", "", "(Optional) Specifies the configuration file path")

	flag.Parse()

	var sshServer ingress.Ingress
	var auth auth.Auth
	var dataStore system.DataStore
	clientChannel := make(chan *Client)
	defer close(clientChannel)

	err := config.BastionConfig.ParseConfig(*configPath)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = logs.Logger.InitLogger(config.BastionConfig.EventsLogFile, config.BastionConfig.SystemLogFile, config.BastionConfig.AsyncEventsLog, config.BastionConfig.AsyncSystemLog)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	logs.Logger.StartLogger()
	defer logs.Logger.StopLogger()

	dataStore, err = system.InitStorage("system")

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = auth.ReadAuthorizedKeysFile(config.BastionConfig.AuthorizedKeysFile)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = sshServer.ConfigSSHServer(auth.AuthorizedKeys, config.BastionConfig.PrivateKeyFile, dataStore)

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	err = sshServer.ConfigTCPListener(config.BastionConfig.ListenAddress + ":" + strconv.Itoa(config.BastionConfig.ListenPort))

	if err != nil {
		fmt.Printf("Error : " + err.Error())
		os.Exit(1)
	}

	go func(sshConfig *ssh.ServerConfig) {
		for {
			c := <-clientChannel

			err = c.handshakeSSH(sshConfig)

			if err != nil {
				log.Print("Failed to handle the TCP conn: ", err)
				continue
			}

			err = c.handleSSHConnexion()

			if err != nil {
				log.Print("Failed to handle the TCP conn: ", err)
				continue
			}

			if c.BackendCommand == "bastion" {

			} else {
				go c.dialBackend()
			}
		}
	}(sshServer.SSHServerConfig)

	for {
		log.Println("Wait for a new connection")
		client := new(Client)

		client.TCPConnexion, err = sshServer.TCPListener.Accept()

		//Create new Client struct and pass it around

		if err != nil {
			log.Printf("failed to accept incoming connection: %v", err)
			continue
		}

		clientChannel <- client
	}
}

func (client *Client) handshakeSSH(sshConfig *ssh.ServerConfig) error {
	// func handshakeSSH(c *net.Conn, sshConfig *ssh.ServerConfig) (*ssh.ServerConn, <-chan ssh.NewChannel, error) {
	// Before use, a handshake must be performed on the incoming
	// net.Conn.
	var reqs <-chan *ssh.Request
	var err error

	client.SSHConnexion, client.sshChan, reqs, err = ssh.NewServerConn(client.TCPConnexion, sshConfig)

	if err != nil {
		log.Print("Failed to handshake: ", err)
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

// func handleSSH(chans <-chan ssh.NewChannel, conn *ssh.ServerConn) {
func (client *Client) handleSSHConnexion() error {
	defer client.SSHConnexion.Close()

	var requests <-chan *ssh.Request
	// Service the incoming Channel channel.
	for newChannel := range client.sshChan {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		var err error
		client.sshCommChan, requests, err = newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
		}
		defer client.sshCommChan.Close()
		break
	}

	for req := range requests {
		if req.Type == "exec" {
			//The request payload is a raw byte array. Its 4 first bytes contain
			//its length so we need to remove them to correctly get the strings
			//TODO set proper limit
			client.RawCommand = string(req.Payload[4:])
			break
		} else if req.Type == "shell" {
			//A shell should not be requested on the bastion
			//This is here to prevent the connexion to hang with a badly formed payload
			client.sshCommChan.Write([]byte("Error : Invalid payload\n"))
			return errors.New("invalid payload")
		}
	}

	if client.RawCommand != "" {
		bc, err := egress.ParseBackendInfo(client.RawCommand)

		//TODO return the correct thing, i was just too lazy to change is for now
		client.BackendCommand = bc.Command
		client.BackendUser = bc.User
		client.BackendHost = bc.Host
		client.BackendPort = bc.Port

		if err != nil {
			errStr := "Error : " + err.Error() + "\n"
			client.sshCommChan.Write([]byte(errStr))
			return errors.New("invalid payload")
		}

		//If no user is provided for the backend, use the one connected to the bastion
		//The username is parsed during the handshake, thus it should not be a problem to
		//use it here directly
		if client.BackendUser == "" {
			client.BackendUser = client.User
		}

	} else {
		client.sshCommChan.Write([]byte("Error : Invalid payload\n"))
		return errors.New("invalid payload")
	}

	return nil
}

func (client *Client) dialBackend() {
	//The user has already been validated during the ssh handshake and should be good
	//We use the connecting user to parse its key
	var err error

	client.SSHKey, err = auth.ParseUserPrivateKey(client.User)

	if err != nil {
		errStr := "Error : " + err.Error() + "\n"
		client.sshCommChan.Write([]byte(errStr))
		return
	}

	bc := egress.BackendConn{
		Command: client.BackendCommand,
		User:    client.BackendUser,
		Host:    client.BackendHost,
		Port:    client.BackendPort,
	}

	// jump to new connection
	err = egress.DialSSH(client.sshCommChan, bc, client.SSHKey)

	if err != nil {
		errStr := "Error : " + err.Error() + "\n"
		client.sshCommChan.Write([]byte(errStr))
	}
}

func (client *Client) runCommand(dataStore system.DataStore) error {
	return nil
}
