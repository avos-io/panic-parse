# Panic-Parse

Panic-parse turns Go panic logs into Sentry events ready for ingestion into the Sentry SDK.

The official Sentry SDK is designed to [capture errors and messages](https://docs.sentry.io/platforms/go/usage/) and as a [panic recovery handler](https://docs.sentry.io/platforms/go/usage/panics/).
However, Go lacks a global panic/exit handler and it doesn't seem like it'll come [any time soon](https://github.com/golang/go/issues/32333).
That means that to catch panics, a recovery handler needs to be added to each goroutine (and main), which is cumbersome, but not impossible.
However it would be easy to forget to add it to future goroutines.

The real kicker though is that you aren't able to recover panics in goroutines created by libraries.
In fact, you can't recover panics created by other packages, so you can't recover panics created in libraries even running on our own threads.

It's therefore better to just let Go crash and upload that to Sentry instead. It catches every case. Even Go recommend scanning for panics externally to the running process.

> For example, perhaps the separate analyzer process could examine the stack trace.

Panic-parse therefore parses a text panic output to convert the stacktrace into the Sentry event struct.
It is designed to be used with a monitor process, such as that provided by [panicwrap](https://github.com/mitchellh/panicwrap/), to catch global panics in the program and report them to Sentry.
