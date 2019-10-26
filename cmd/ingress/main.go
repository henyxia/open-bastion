package main

import (
    "fmt"
    "golang.org/x/crypto/ssh"
    "io"
    "io/ioutil"
    "log"
    "net"
)

func main() {
    // Build SSH client configuration
    authorizedKeysMap := map[string]bool{}

    pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAII+XathO8hD4diyvyT4N0FgsuVqnkuPkA1q0DcGhpDQs henyxia@phy0"))
    if err != nil {
        log.Fatal(err)
    }

    authorizedKeysMap[string(pubKey.Marshal())] = true

    // An SSH server is represented by a ServerConfig, which holds
    // certificate details and handles authentication of ServerConns.
    config := &ssh.ServerConfig{
        PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
            if authorizedKeysMap[string(pubKey.Marshal())] {
                return &ssh.Permissions{
                    // Record the public key used for authentication.
                    Extensions: map[string]string{
                        "pubkey-fp": ssh.FingerprintSHA256(pubKey),
                    },
                }, nil
            }
            return nil, fmt.Errorf("unknown public key for %q", c.User())
        },
    }

    privateBytes, err := ioutil.ReadFile("/home/henyxia/.ssh/test-bastion")
    if err != nil {
        log.Fatal("Failed to load private key: ", err)
    }

    private, err := ssh.ParsePrivateKey(privateBytes)
    if err != nil {
        log.Fatal("Failed to parse private key: ", err)
    }

    config.AddHostKey(private)

    //Listen for incoming SSH conn

    // Once a ServerConfig has been configured, connections can be
    // accepted.
    log.Println("listen for incoming connections")
    listener, err := net.Listen("tcp", "0.0.0.0:2022")
    if err != nil {
        log.Fatal("failed to listen for connection: ", err)
    }
    log.Println("accept new connection")
    client, err := listener.Accept()
    if err != nil {
        log.Fatal("failed to accept incoming connection: ", err)
    }

    // Before use, a handshake must be performed on the incoming
    // net.Conn.
    hsConn, chans, reqs, err := ssh.NewServerConn(client, config)
    if err != nil {
        log.Fatal("failed to handshake: ", err)
    }
    log.Printf("logged in with key %s", hsConn.Permissions.Extensions["pubkey-fp"])

    // The incoming Request channel must be serviced.
    go ssh.DiscardRequests(reqs)

    // Service the incoming Channel channel.
    for newChannel := range chans {
        log.Printf("channel opened (type=%s)", newChannel.ChannelType())
        if newChannel.ChannelType() != "session" {
            newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
            continue
        }
        channel, _, err := newChannel.Accept()
        if err != nil {
            log.Fatalf("Could not accept channel: %v", err)
        }

        // jump to new connection
        dialSSH(channel)
    }
}

func dialSSH(channel ssh.Channel) {
    server := "127.0.0.1"
    port := "22"

    server = server + ":" + port
    user := "henyxia"
    p := "top-secreto"

    var pass = string(p)
    config := &ssh.ClientConfig{
        User: user,
        Auth: []ssh.AuthMethod{
            // ClientAuthPassword wraps a ClientPassword implementation
            // in a type that implements ClientAuth.
            ssh.Password(pass),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }
    sshConn, err := ssh.Dial("tcp", server, config)
    if err != nil {
        panic("Failed to dial: " + err.Error())
    }
    defer sshConn.Close()

    // Each ClientConn can support multiple interactive sessions,
    // represented by a Session.
    session, err := sshConn.NewSession()
    if err != nil {
        panic("Failed to create session: " + err.Error())
    }
    defer session.Close()

    stdin, err := session.StdinPipe()
    if err != nil {
        fmt.Errorf("Unable to setup stdin for session: %v", err)
    }
    go io.Copy(stdin, channel)

    stdout, err := session.StdoutPipe()
    if err != nil {
        fmt.Errorf("Unable to setup stdout for session: %v", err)
    }
    go io.Copy(channel, stdout)

    // Set up terminal modes
    modes := ssh.TerminalModes{
        ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
        ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
    }

    // Request pseudo terminal
    log.Println("request pseudo terminal")
    if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
        log.Fatalf("request for pseudo terminal failed: %s", err)
    }

    // Start remote shell
    log.Println("start shell")
    if err := session.Shell(); err != nil {
        log.Fatalf("failed to start shell: %s", err)
    }

    log.Println("waiting command")
    err = session.Wait()
    if err != nil {
        if err, ok := err.(*ssh.ExitError); ok {
            fmt.Errorf("commad exited with: %v", err)
        } else {
            fmt.Errorf("failed to start command: %v", err)
        }
    }
    log.Println("end")
}
