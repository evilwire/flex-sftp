package main

import (
	"github.com/evilwire/flex-sftp"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	echo "github.com/labstack/echo"
)

type App struct {
	web *echo.Echo
	sftp *flex.SFTPServer
}

func NewApp() *App {
	return &App{}
}

const (
	healthcheckEndpoint = "/health"
	metaEndpoint = "/meta"
	metricsEndpoint = "/metrics"
)

func (app *App) Setup() (err error) {
	// add a health-check endpoint
	webServer := echo.New()
	webServer.Add(echo.GET, healthcheckEndpoint, func(c echo.Context) error {
		return c.JSON(200, struct {
			Status string `json:"status"`
		}{
			Status: "ok",
		})
	})

	webServer.GET(metricsEndpoint, echo.WrapHandler(promhttp.Handler()))

	sftpServer := flex.NewSFTPServer(flex.Config{ListenerCount: 5})
	if err = sftpServer.SetupEventLoop(); err != nil {
		return
	}

	app.web = webServer
	app.sftp = sftpServer
	return
}

func (app *App) Run() error {
	errChan := make(chan error)
	go func() {
		errChan <- app.sftp.ListenAndServe("0.0.0.0:2222")
	}()

	go func() {
		errChan <- app.web.Start(":8080")
	}()

	return <- errChan
}
