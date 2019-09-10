package main

import (
	"flag"

	"github.com/golang/glog"
)

func main() {
	flag.Parse()

	server := SFTPServer{
		config: Config{ListenerCount: 5},
	}
	if err := server.setupEventLoop(); err != nil {
		panic(err)
	}

	glog.Infof("Starting test server...")
	panic(server.ListenAndServe("0.0.0.0:2022"))
}
