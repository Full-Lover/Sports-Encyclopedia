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
	if len(args) != 1 {
		return errors.New("usage: sports-encyclopedia web|migrate|publish-preview")
	}
	if args[0] == "migrate" || args[0] == "publish-preview" {
		return runPublicationCommand(args[0])
	}
	if args[0] != "web" {
		return errors.New("usage: sports-encyclopedia web|migrate|publish-preview")
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
	preview, err := publishedatlas.PreviewMapDocument()
	if err != nil {
		return fmt.Errorf("load preview map: %w", err)
	}
	previous, err := publishedatlas.Preview0006MapDocument()
	if err != nil {
		return fmt.Errorf("load previous preview map: %w", err)
	}

	reader := publishedatlas.Reader(publishedatlas.NewMemoryReader(
		preview,
		previous,
		publishedatlas.Preview0005MapDocument(),
		publishedatlas.Preview0004MapDocument(),
		publishedatlas.Preview0003MapDocument(),
		publishedatlas.Preview0002MapDocument(),
		publishedatlas.Preview0001MapDocument(),
	))
	if os.Getenv("SPORTS_DB_DSN") != "" {
		db, err := openPublicationDB()
		if err != nil {
			return err
		}
		defer db.Close()
		reader = publishedatlas.NewMySQLPublicationStore(db)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           webexperience.New("web/dist", reader),
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
