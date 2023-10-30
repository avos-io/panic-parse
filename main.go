package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/avos-io/gocrash/panicparse"
	sentry "github.com/getsentry/sentry-go"
	"github.com/mitchellh/panicwrap"
)

const (
	wrap = true // Could be set with a command line flag or environment variable
)

func main() {
	sentry.Init(sentry.ClientOptions{
		Dsn: "your dsn here",
	})
	defer sentry.Flush(time.Second * 5)

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
	oldmain()
}

func oldmain() {
	// Let's say we panic
	//panic("oh shucks")

	// Or we can panic with a signal
	var a *int
	*a = 1
}

func panicHandler(output string) {
	event := panicparse.Parse(strings.NewReader(output))
	event.Extra["panic"] = output

	json, _ := json.MarshalIndent(event, "", "  ")
	fmt.Printf("panic report: %v\n", string(json))

	id := sentry.CaptureEvent(event)
	fmt.Printf("sentry event id: %v\n", id)

	os.Exit(1)
}
