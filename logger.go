package yts

import (
	"log"
	"os"
	"sync"
)

type debugWriter struct {
	debug bool
	mu    sync.Mutex
}

func (dw *debugWriter) setDebug(debug bool) {
	dw.debug = debug
}

func (dw *debugWriter) Write(p []byte) (int, error) {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	if !dw.debug {
		return 0, nil
	}

	return os.Stdout.Write(p)
}

type logger struct {
	log.Logger
	debugWriter
}

func newLogger() *logger {
	l := &logger{}
	l.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	l.SetPrefix("(yflicks-yts): ")
	l.SetOutput(&l.debugWriter)
	return l
}
