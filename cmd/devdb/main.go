// devdb runs a throwaway embedded PostgreSQL for local development without
// Docker: go run ./cmd/devdb  (listens on :5433, Ctrl+C to stop).
// The Docker Compose setup is the intended deployment; this is a convenience.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func main() {
	runtimeDir := filepath.Join(os.TempDir(), "spacedchess-devdb")
	db := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(5433).
		RuntimePath(runtimeDir).
		DataPath(filepath.Join(runtimeDir, "data")).
		Database("spacedchess"))
	if err := db.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}
	fmt.Println("devdb ready: postgres://postgres:postgres@localhost:5433/spacedchess?sslmode=disable")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	if err := db.Stop(); err != nil {
		fmt.Fprintln(os.Stderr, "stop:", err)
		os.Exit(1)
	}
}
