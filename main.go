package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/avos-io/gocrash/panicparse"
	"github.com/mitchellh/panicwrap"
)

var sentry = panicparse.FromDNS("your dsn here")

func main() {
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		// Something went wrong setting up the panic wrapper. Unlikely,
		// but possible.
		panic(err)
	}

	// If exitStatus >= 0, then we're the parent process and the panicwrap
	// re-executed ourselves and completed. Just exit with the proper status.
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	// Otherwise, exitStatus < 0 means we're the child. Continue executing as
	// normal...

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

	id, err := sentry.SendCrashReport(event)
	if err != nil {
		fmt.Printf("send crash report failed: %v\n", err)
	} else {
		fmt.Printf("sentry event id: %v\n", id)
	}

	os.Exit(1)
}
