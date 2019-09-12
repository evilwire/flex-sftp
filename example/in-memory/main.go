package main

import (
	"flag"
	"github.com/golang/glog"
)


func main() {
	flag.Parse()

	app := NewApp()
	err := app.Setup()
	if err != nil {
		panic(err)
	}

	glog.Infof("Starting test server...")
	panic(app.Run())
}
