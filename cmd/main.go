package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	panicparse "github.com/avos-io/panic-parse"
	sentry "github.com/getsentry/sentry-go"
	"github.com/mitchellh/panicwrap"
)

const (
	wrap = true // Could be set with a command line flag or environment variable
)

var (
	sentryTimeout = 5 * time.Second
)

func main() {
	if wrap {
		exitStatus, err := panicwrap.BasicWrap(panicHandler)
		if err != nil {
			panic(err)
		}

		// If exitStatus >= 0, then we're the parent process and the panicwrap
		// re-executed ourselves and completed. Just exit with the proper status.
		if exitStatus >= 0 {
			os.Exit(exitStatus)
		}
	}

	// Else we're the child so execute our real main function
	oldMain()
}

func oldMain() {
	// You can use a Sentry like normal here if you'd like
	initSentry(false)

	// Let's say we panic
	//panic("oh shucks")

	// Or we can panic with a signal
	var a *int
	*a = 1
}

func initSentry(sync bool) {
	var transport sentry.Transport

	if sync {
		syncTransport := sentry.NewHTTPSyncTransport()
		syncTransport.Timeout = sentryTimeout
		transport = syncTransport
	}

	sentry.Init(sentry.ClientOptions{
		Dsn:       "your dsn here",
		Transport: transport,
	})
}

func panicHandler(output string) {
	initSentry(true)

	event := panicparse.Parse(strings.NewReader(output))
	event.Extra["panic"] = output

	json, _ := json.MarshalIndent(event, "", "  ")
	fmt.Printf("panic report: %v\n", string(json))

	id := sentry.CaptureEvent(event)
	fmt.Printf("sentry event id: %v\n", *id)

	os.Exit(1)
}
