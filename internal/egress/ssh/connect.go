package ssh

import (
	"io"
	"errors"
	"strconv"

	"golang.org/x/crypto/ssh"

	logger "github.com/open-bastion/open-bastion/internal/logger"
	"github.com/open-bastion/open-bastion/internal/egress/backend"
)
// Connect contact the destination backend server
func Connect(channel ssh.Channel, bc backend.Conn, signer ssh.Signer) error {
	pcb := func() (string, error) {
		return "", nil
	}

	var authMethods = []ssh.AuthMethod{ssh.PasswordCallback(pcb)}

	if signer != nil {
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	config := &ssh.ClientConfig{
		User: bc.User,
		Auth: authMethods,
		//This should be replaced by HostKeyCallBack and use a mecanism to
		//verify the backend host key
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshConn, err := ssh.Dial("tcp", bc.Host+":"+strconv.Itoa(bc.Port), config)
	if err != nil {
		return errors.New("Error dialing backend : " + err.Error())
	}
	defer sshConn.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := sshConn.NewSession()
	if err != nil {
		return errors.New("Error creating new session : " + err.Error())
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return errors.New("Error getting session stdin : " + err.Error())
	}

	go func() {
		copy(stdin, channel, nil)
	}()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return errors.New("Error getting session stdout : " + err.Error())
	}

	go func() {
		copy(channel, stdout, nil)
	}()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return errors.New("Error requesting pseudo terminal : " + err.Error())
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		return errors.New("Error starting shell : " + err.Error())
	}

	logger.Debugf("Shell started, waiting command")
	err = session.Wait()
	if err != nil {
		if err, ok := err.(*ssh.ExitError); ok {
			logger.Debugf("Command exited with: %v", err)
		} else {
			logger.Debugf("Failed to start command: %v", err)
		}
	}

	return nil
}

// This function is a reimplementation of the io.Copy function but take a chan where it also write
// the data copied
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
