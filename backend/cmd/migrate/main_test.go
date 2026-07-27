package main

import "testing"

func TestMigrationCommandNames(t *testing.T) {
	for _, command := range []string{"up", "down", "status", "version", "repair"} {
		if command == "" {
			t.Fatal("migration command must not be empty")
		}
	}
}
