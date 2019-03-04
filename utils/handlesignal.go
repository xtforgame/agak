package utils

import (
	"os"
	"os/signal"
	"syscall"
)

func GetSIGINTandSIGTERMChan() chan os.Signal {
	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}

func HandleSIGINTandSIGTERM() {
	// Handle SIGINT and SIGTERM.
	ch := GetSIGINTandSIGTERMChan()
	<-ch
}
