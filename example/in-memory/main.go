package main

import (
	"flag"
	"github.com/golang/glog"

	flex "github.com/evilwire/flex-sftp"
)


func main() {
	flag.Parse()

	server := flex.NewSFTPServer(flex.Config{ListenerCount: 5})
	if err := server.SetupEventLoop(); err != nil {
		panic(err)
	}

	glog.Infof("Starting test server...")
	panic(server.ListenAndServe("0.0.0.0:2022"))
}
