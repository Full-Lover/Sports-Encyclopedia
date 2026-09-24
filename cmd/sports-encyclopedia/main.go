package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
	"github.com/full-lover/sports-encyclopedia/internal/webexperience"
)

const defaultAddress = ":8080"

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) != 1 || args[0] != "web" {
		return errors.New("usage: sports-encyclopedia web")
	}

	for _, asset := range []string{"web/dist/assets/app.js", "web/dist/assets/app.css"} {
		if _, err := os.Stat(asset); err != nil {
			return fmt.Errorf("frontend asset %q is unavailable; run 'npm --prefix web run build': %w", asset, err)
		}
	}

	address := os.Getenv("APP_ADDR")
	if address == "" {
		address = defaultAddress
	}

	server := &http.Server{
		Addr: address,
		Handler: webexperience.New(
			"web/dist",
			publishedatlas.NewMemoryReader(
				publishedatlas.PreviewMapDocument(),
				publishedatlas.Preview0002MapDocument(),
				publishedatlas.Preview0001MapDocument(),
			),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("web server listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve web: %w", err)
	}
	return nil
}
