// Package logging configures the process-wide logrus logger.
package logging

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

// Setup configures log level and output format.
// When jsonOutput is true, structured JSON logs are emitted (useful for CI).
func Setup(verbose bool, jsonOutput bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetOutput(os.Stderr)

	if jsonOutput {
		log.SetFormatter(&log.JSONFormatter{})
		return
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
		DisableColors:   false,
	})
}

// Discard silences logging (used by tests).
func Discard() {
	log.SetOutput(io.Discard)
}
