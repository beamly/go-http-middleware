package middleware

import (
	"encoding/json"
	"log"
	"os"
)

// defaultLogger implements middleware.Loggable
// and, by default, spits some json to STDOUT
//
// This is the simplest log there is.
type defaultLogger struct {
	output *log.Logger
}

func newDefaultLogger() defaultLogger {
	return defaultLogger{
		output: log.New(os.Stdout, "", 0),
	}
}

// Log will spit out a LogEntry marshaled to json
// to STDOUT
func (dl defaultLogger) Log(l LogEntry) {
	lOut, err := json.Marshal(l)

	if err == nil {
		dl.output.Print(string(lOut))
	} else {
		dl.output.Printf("error marshaling log data: %q", err)
	}
}
