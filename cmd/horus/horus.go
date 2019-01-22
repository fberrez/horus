package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fberrez/horus/api"
	"github.com/fberrez/horus/lifx"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	defaultPort = "2020"
)

func init() {
	env := os.Getenv("ENVIRONMENT")

	if env == "" || env == "DEV" {
		// Log as the default ASCII formatter.
		log.SetFormatter(&log.TextFormatter{})

		// Output to stdout instead of the default stderr
		log.SetOutput(os.Stdout)

		// Log all messages.
		log.SetLevel(log.DebugLevel)
	} else if env == "PROD" {
		// Log as JSON instead of the default ASCII formatter.
		log.SetFormatter(&log.JSONFormatter{})

		// Output to stdout instead of the default stderr
		log.SetOutput(os.Stdout)

		// Only log the warning severity or above.
		log.SetLevel(log.WarnLevel)

		// Sets mode of the API on release mode.
		gin.SetMode(gin.ReleaseMode)
	}
}

func main() {
	err := lifx.LoadProducts()
	if err != nil {
		panic(err)
	}

	api, err := api.New()
	if err != nil {
		panic(err)
	}

	srv := &http.Server{
		Addr:    listenPort(os.Getenv("SERVER_PORT")),
		Handler: api,
	}

	// Runs the server
	go func() {
		if errListen := srv.ListenAndServe(); errListen != nil && errListen != http.ErrServerClosed {
			panic(errListen)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGTERM)
	signal.Notify(quit, syscall.SIGINT)

	// Wait for a SIGTERM or SIGINT
	<-quit
	if err := srv.Shutdown(context.Background()); err != nil {
		panic(err)
	}

	log.Info("Graceful shutdown")
	os.Exit(0)
}

// listenPort returns a port number according if it has been defined or not.
func listenPort(port string) string {
	if len(port) == 0 {
		port = defaultPort
	}

	log.Infof("Server running on :%s", port)

	return ":" + port
}
