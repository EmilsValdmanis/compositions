package database

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestMigrationLoggerVerbose(t *testing.T) {
	if !(migrationLogger{}).Verbose() {
		t.Fatal("migrationLogger.Verbose() = false; want true")
	}
}

func TestMigrationLoggerPrintf(t *testing.T) {
	var buffer bytes.Buffer
	originalWriter := log.Writer()
	defer log.SetOutput(originalWriter)
	log.SetOutput(&buffer)

	(migrationLogger{}).Printf("applied %s", "000001_create_users")

	if got := buffer.String(); !strings.Contains(got, "applied 000001_create_users") {
		t.Fatalf("migrationLogger.Printf() output = %q; want log line", got)
	}
}
