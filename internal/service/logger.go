package service

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Logger struct {
	mu sync.Mutex
}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) log(level, msg string, fields map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := map[string]interface{}{
		"level": level,
		"time":  time.Now().UTC().Format(time.RFC3339),
		"msg":   msg,
	}

	if fields != nil {
		entry["fields"] = fields
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"level":"error","time":"%s","msg":"logger marshal failed","fields":{"error":"%s"}}`+"\n",
			time.Now().UTC().Format(time.RFC3339), err.Error())
		return
	}

	fmt.Fprintln(os.Stderr, string(data))
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log("info", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log("error", msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log("warn", msg, fields)
}
