package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	ListenerCount uint `env:"LISTENER_COUNT"`
}

type SFTPConnectionListener struct {
	config   *ssh.ServerConfig
	Handlers sftp.Handlers
}

func (listener *SFTPConnectionListener) handShake(connection net.Conn) (channels <-chan ssh.NewChannel, err error) {
	serverConn, channels, reqs, err := ssh.NewServerConn(connection, listener.config)
	if err != nil {
		glog.Errorf("failed to handshake: %v", err)
		return
	}

	glog.Infof("login detected: %s", serverConn.User())
	glog.Info("SSH server established\n")

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)
	return
}

func (listener *SFTPConnectionListener) replySubsystemReq(requests <-chan *ssh.Request) {
	for req := range requests {
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
		err := req.Reply(ok, nil)
		if err != nil {
			glog.Errorf("error replying to request: %v", err)
		}
	}
}

func (listener *SFTPConnectionListener) ProcessNewChannels(newChannel ssh.NewChannel) (err error) {
	glog.Infof("Incoming channel: %s\n", newChannel.ChannelType())
	if newChannel.ChannelType() != "session" {
		if rejectErr := newChannel.Reject(ssh.UnknownChannelType, "unknown channel type"); rejectErr != nil {
			glog.Errorf("error encountered in rejecting new channel request: %v", rejectErr)
		}
		return errors.Errorf("unknown channel type: %s", newChannel.ChannelType())
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		glog.Errorf("could not accept channel: %v", err)
		return
	}

	go listener.replySubsystemReq(requests)
	glog.Info("Channel accepted\n")

	server := sftp.NewRequestServer(channel, listener.Handlers)
	if err := server.Serve(); err == io.EOF {
		err = server.Close()
		glog.Info("sftp client exited session.")
	} else if err != nil {
		glog.Errorf("sftp server completed with error: %v", err)
	}
	return
}

func (listener *SFTPConnectionListener) Listen(connection net.Conn) (err error) {
	newChannels, err := listener.handShake(connection)
	if err != nil {
		return
	}

	for newChannel := range newChannels {
		err = listener.ProcessNewChannels(newChannel)
		if err != nil {
			glog.Errorf("encountered error while processing channels: %v", err)
			continue
		}
	}

	return
}

type ConnectionRequest struct {
	Connection net.Conn
	Timestamp  time.Time
}

type SFTPServer struct {
	config Config
	connections chan ConnectionRequest
}

func (server *SFTPServer) setupEventLoop() {
	server.connections = make(chan ConnectionRequest, server.config.ListenerCount)

	// make a load of connection listeners
	listener := SFTPConnectionListener{
		config: &ssh.ServerConfig{
			PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
				// Should use constant-time compare (or better, salt+hash) in
				// a production setting.
				glog.Infof("Login: %s\n", c.User())
				if c.User() == "testuser" && string(pass) == "tiger" {
					return nil, nil
				}
				return nil, fmt.Errorf("password rejected for %q", c.User())
			},
		},

		Handlers: sftp.InMemHandler(),
	}

	for i := 0; i < int(server.config.ListenerCount); i++ {
		// listen to the connections
		go func(connRequests <-chan ConnectionRequest) {
			for connReq := range connRequests {
				// record the lag time
				err := listener.Listen(connReq.Connection)
				if err != nil {
					glog.Errorf("error processing connection: %v", err)
				}
			}
		}(server.connections)
	}
}

func (server *SFTPServer) ListenAndServe(addr string) (err error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		glog.Errorf("Failed to listen for connection: %v", err)
		return
	}

	glog.Infof("Listening on %v\n", listener.Addr())
	defer func() {
		r := recover()
		if r != nil {
			if rErr, ok := r.(error); ok {
				err = rErr
			}
		}
	}()

	for {
		connReq, err := listener.Accept()
		if err != nil {
			glog.Errorf("error accepting new connection: %v", err)
		}
		server.connections <- ConnectionRequest{
			Connection: connReq,
			Timestamp:  time.Now(),
		}
	}
}
