package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"

	"github.com/golang/glog"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func main() {
	flag.Parse()

	server := SFTPServer{
		config: Config{ListenerCount: 5},
	}
	server.setupEventLoop()
	panic(server.ListenAndServe("0.0.0.0:2022"))

}

func oldMain() {
	flag.Parse()

	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in
			// a production setting.
			glog.Infof("Login: %s\n", c.User())
			if c.User() == "testuser" && string(pass) == "tiger" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	glog.Info("Allocating a new in-memory handler")
	root := sftp.InMemHandler()

	privateBytes, err := ioutil.ReadFile("/usr/keys/id_rsa")
	if err != nil {
		glog.Fatal("Failed to load private key", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		glog.Fatal("Failed to parse private key", err)
	}
	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2022")
	if err != nil {
		glog.Fatal("failed to listen for connection", err)
	}
	glog.Infof("Listening on %v\n", listener.Addr())

	wg := sync.WaitGroup{}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			for {
				nConn, err := listener.Accept()
				if err != nil {
					glog.Fatal("failed to accept incoming connection", err)
				}

				// Before use, a handshake must be performed on the incoming net.Conn.
				sconn, chans, reqs, err := ssh.NewServerConn(nConn, config)
				if err != nil {
					glog.Fatal("failed to handshake", err)
				}
				glog.Infof("login detected: %s", sconn.User())
				glog.Info("SSH server established\n")

				// The incoming Request channel must be serviced.
				go ssh.DiscardRequests(reqs)

				// Service the incoming Channel channel.
				for newChannel := range chans {
					// Channels have a type, depending on the application level
					// protocol intended. In the case of an SFTP session, this is "subsystem"
					// with a payload string of "<length=4>sftp"
					glog.Infof("Incoming channel: %s\n", newChannel.ChannelType())
					if newChannel.ChannelType() != "session" {
						newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
						glog.Infof("Unknown channel type: %s\n", newChannel.ChannelType())
						continue
					}

					channel, requests, err := newChannel.Accept()
					if err != nil {
						glog.Fatal("could not accept channel.", err)
					}
					glog.Info("Channel accepted\n")

					// Sessions have out-of-band requests such as "shell",
					// "pty-req" and "env".  Here we handle only the
					// "subsystem" request.
					go func(in <-chan *ssh.Request) {
						for req := range in {
							glog.Infof("Request: %v\n", req.Type)
							ok := false
							switch req.Type {
							case "subsystem":
								glog.Infof("Subsystem: %s\n", req.Payload[4:])
								if string(req.Payload[4:]) == "sftp" {
									ok = true
								}
							}
							glog.Infof(" - accepted: %v\n", ok)
							req.Reply(ok, nil)
						}
					}(requests)

					server := sftp.NewRequestServer(channel, root)
					if err := server.Serve(); err == io.EOF {
						server.Close()
						glog.Info("sftp client exited session.")
					} else if err != nil {
						glog.Fatal("sftp server completed with error:", err)
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
