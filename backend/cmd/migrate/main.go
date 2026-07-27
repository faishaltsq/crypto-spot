package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/example/crypto-spot-signal/internal/migration"
)

func main() {
	if len(os.Args) != 2 {
		fail("usage: migrate <up|down|status|version|repair>")
	}
	cfg := migration.DefaultConfig(os.Getenv("DATABASE_URL"))
	var err error
	switch os.Args[1] {
	case "up":
		_, err = migration.Up(context.Background(), cfg)
	case "down":
		_, err = migration.Down(context.Background(), cfg)
	case "status", "version":
		var status migration.Status
		status, err = migration.StatusOf(context.Background(), cfg)
		if err == nil {
			fmt.Printf("version=%d latest=%d dirty=%t\n", status.Version, status.Latest, status.Dirty)
		}
	case "repair":
		_, err = migration.Repair(context.Background(), cfg)
	default:
		fail("usage: migrate <up|down|status|version|repair>")
	}
	if err != nil {
		fail("migration %s failed: %v", os.Args[1], err)
	}
}

func fail(format string, args ...any) {
	log.Printf(format, args...)
	os.Exit(1)
}
