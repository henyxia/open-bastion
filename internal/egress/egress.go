package egress

import (
	"context"
	"errors"
	"github.com/open-bastion/open-bastion/internal/datastore"
	"github.com/open-bastion/open-bastion/internal/obclient"
	"io"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/open-bastion/open-bastion/internal/logger"
)

//EstablishSSHConnection takes a client connected with SSH to the bastion and tries to get its information from the
//datastore to establish a connection to the backend.
func EstablishSSHConnection(ctx context.Context, client *obclient.Client, dataStore datastore.DataStore) {
	var err error
	//The user has already been validated during the ssh handshake and should be good
	//We use the connecting user to parse its key
	client.SSHKey, err = dataStore.GetUserEgressPrivateKeySigner(client.User)

	logger.UpdateClientLogCtx(ctx, client)

	if err != nil {
		_, _ = client.SshCommChan.Write([]byte("error accessing credentials"))

		logger.ErrorWithCtxWithErr(ctx, err, "authenticated user could not access his egress private key")
	}

	//TODO add proper cancel func to clean up and display an error
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*1000)
	defer cancel()

	err = DialBackend(ctx, client)

	if err != nil {
		logger.WarnWithCtxWithErr(ctx, err, "error dialing backend")
	}
}

//DialBackend takes the context and a client pointer with a already established SSH connection. It then tries to
//connect to an SSH backend with the client information
func DialBackend(ctx context.Context, client *obclient.Client) error {
	// jump to new connection
	err := DialSSH(ctx, client)

	if err != nil {
		errStr := "Error : " + err.Error() + "\n"
		_, _ = client.SshCommChan.Write([]byte(errStr))
		return err
	}

	return nil
}

// DialSSH contact the destination backend server
//func DialSSH(ctx context.Context, channel ssh.Channel, bc BackendConn, signer ssh.Signer) error {
func DialSSH(ctx context.Context, client *obclient.Client) error {
	pcb := func() (string, error) {
		return "", nil
	}

	var authMethods = []ssh.AuthMethod{ssh.PasswordCallback(pcb)}

	if client.SSHKey != nil {
		authMethods = append(authMethods, ssh.PublicKeys(client.SSHKey))
	}

	timeout := time.Duration(0)
	dl, ok := ctx.Deadline()

	if ok {
		timeout = time.Until(dl)
	}

	config := &ssh.ClientConfig{
		User: client.User,
		Auth: authMethods,
		//TODO This should be replaced by HostKeyCallBack and use a mechanism to
		//verify the backend host key
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	sshConn, err := ssh.Dial("tcp", client.BackendHost+":"+strconv.Itoa(client.BackendPort), config)
	if err != nil {
		return errors.New("error dialing backend : " + err.Error())
	}

	defer func() {
		if err := sshConn.Close(); err != nil {
			logger.WarnWithCtxWithErr(ctx, err, "error closing SSH connection")
		}
	}()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := sshConn.NewSession()

	if err != nil {
		return errors.New("error creating new session : " + err.Error())
	}

	defer func() {
		if err := session.Close(); err != nil {
			logger.WarnWithCtxWithErr(ctx, err, "error closing session")
		}
	}()

	stdin, err := session.StdinPipe()

	if err != nil {
		return errors.New("error getting session stdin : " + err.Error())
	}

	defer func() {
		if err := client.SshCommChan.CloseWrite(); err != nil {
			logger.WarnWithCtxWithErr(ctx, err, "error closing write of SSH comm chan")
		}

		if err := stdin.Close(); err != nil {
			logger.WarnWithCtxWithErr(ctx, err, "error closing session stdin pipe")
		}
	}()

	go func() {
		_, _ = copy(stdin, client.SshCommChan, nil)
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.New("Error getting session stdout : " + err.Error())
	}

	go func() {
		_, _ = copy(client.SshCommChan, stdout, nil)
	}()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return errors.New("error requesting pseudo terminal : " + err.Error())
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		return errors.New("Error starting shell : " + err.Error())
	}

	logger.Debugf("shell started, waiting command")
	err = session.Wait()
	if err != nil {
		if err, ok := err.(*ssh.ExitError); ok {
			logger.Debugf("command exited with: %v", err)
		} else {
			logger.Debugf("failed to start command: %v", err)
		}
	}
	logger.InfoWithCtx(ctx, "client disconnected")

	return nil
}

// copy is a reimplementation of the io.Copy function but takes a chan where it also write
// the data copied
//
// io.Copy copies from src to dst until either EOF is reached
// on src or an error occurs. It returns the number of bytes
// copied and the first error encountered while copying, if any.
//
// A successful Copy returns err == nil, not err == EOF.
// Because Copy is defined to read from src until EOF, it does
// not treat an EOF from Read as an error to be reported.
//
// If src implements the WriterTo interface,
// the copy is implemented by calling src.WriteTo(dst).
// Otherwise, if dst implements the ReaderFrom interface,
// the copy is implemented by calling dst.ReadFrom(src).
func copy(dst io.Writer, src io.Reader, log chan []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)

	for {
		nr, er := src.Read(buf)

		if log != nil {
			log <- buf[0:nr]
		}

		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
