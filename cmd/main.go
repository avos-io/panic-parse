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
	wrap = true            // Could be set with a command line flag or environment variable
	dsn  = "your dsn here" // Could be set with an environment variable or secret

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
	// Use Async transport to avoid blocking the panic recovery/err handler while
	// the event is sent to Sentry
	cleanup := initSentry(false)
	defer cleanup()

	// Let's say we panic
	//panic("oh shucks")

	// Or we can panic with a signal
	var a *int
	*a = 1
}

// Initialise the Sentry SDK
//
// If sync is true, Sentry events are sent synchronously. This is useful for
// ensuring that the event is sent before the process exits. However, it can
// cause the process to hang if the Sentry server is down or unreachable.
//
// The cleanpu function can be ignored if sync is false.
func initSentry(sync bool) func() {
	var transport sentry.Transport

	cleanup := func() {
		sentry.Flush(sentryTimeout)
	}

	if sync {
		syncTransport := sentry.NewHTTPSyncTransport()
		syncTransport.Timeout = sentryTimeout
		transport = syncTransport
		cleanup = func() {}
	}

	sentry.Init(sentry.ClientOptions{
		Dsn:       dsn,
		Transport: transport,
	})

	return cleanup
}

func panicHandler(output string) {
	// Use sync transport since we're dying anyway
	initSentry(true)

	event := panicparse.Parse(strings.NewReader(output))
	event.Extra["panic"] = output

	json, _ := json.MarshalIndent(event, "", "  ")
	fmt.Printf("panic report: %v\n", string(json))

	id := sentry.CaptureEvent(event)
	fmt.Printf("sentry event id: %v\n", *id)

	os.Exit(1)
}
