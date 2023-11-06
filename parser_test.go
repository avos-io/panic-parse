package panicparse_test

import (
	"strings"
	"testing"

	panicparse "github.com/avos-io/panic-parse"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicParse(t *testing.T) {
	for name, tc := range testCases {
		c := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := panicparse.Parse(strings.NewReader(c.Data))

			compareEvents(t, c.Result, event)
		})
	}
}

var testCases = map[string]struct {
	Data   string
	Result *sentry.Event
}{
	"segfault": {
		Data: `panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0xffffffff addr=0x0 pc=0x20314]

goroutine 1 [running]:
panic(0x112c00, 0x1040a038)
/usr/local/go/src/runtime/panic.go:500 +0x720
main.main()
/tmp/sandbox675251439/main.go:23 +0x314`,
		Result: &sentry.Event{
			Message: "invalid memory address or nil pointer dereference",
			Exception: []sentry.Exception{{
				Type:     "runtime error",
				Value:    "invalid memory address or nil pointer dereference",
				ThreadID: "1",
				Mechanism: &sentry.Mechanism{
					Type:        "signal",
					Data:        map[string]interface{}{"signal": "SIGSEGV", "code": "0xffffffff", "relevant_address": "0x0", "program_counter": "0x20314"},
					Description: "segmentation violation",
					Handled:     new(bool),
				},
			}},
			Threads: []sentry.Thread{{
				ID: "1",
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{
						{
							Package:  "main",
							Function: "main",
							Filename: "/tmp/sandbox675251439/main.go",
							Lineno:   23,
							InApp:    true,
						},
						{
							Function: "panic",
							Filename: "/usr/local/go/src/runtime/panic.go",
							Lineno:   500,
							InApp:    false,
						},
					},
				},
			}},
			Level: "fatal",
		},
	},
	"panic": {
		Data: `panic: oh my god

goroutine 86 [running]:
github.com/avos-io/iona/lindisfarne/internal/endpoints.(*Server).ReportDynamicInfo(0x58?, {0x1419348, 0xc0004924b0}, 0x2?)
	/home/jon/source/iona/lindisfarne/internal/endpoints/endpoints.go:868 +0x386
github.com/avos-io/protorepo/gen/go/lindisfarne._Lindisfarne_ReportDynamicInfo_Handler.func1({0x1419348, 0xc0004924b0}, {0x1162420?, 0xc000248000})
	/home/jon/source/iona/protorepo/gen/go/lindisfarne/lindisfarne_grpc.pb.go:312 +0x78
github.com/avos-io/iona/endpointauth.(*Interceptor).Unary.func1({0x1419348?, 0xc0003e3ce0?}, {0x1162420, 0xc000248000}, 0xc000201c00, 0xc000305e18)
	/home/jon/source/iona/endpointauth/interceptor.go:52 +0x1a8
google.golang.org/grpc.getChainUnaryHandler.func1({0x1419348, 0xc0003e3ce0}, {0x1162420, 0xc000248000})
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:1179 +0xb9
github.com/avos-io/iona/lindisfarne/internal/endpoints.(*Interceptor).Unary.func1({0x1419348, 0xc00035b800}, {0x1162420, 0xc000248000}, 0xc000201c00?, 0xc00027f4c0)
	/home/jon/source/iona/lindisfarne/internal/endpoints/interceptor.go:60 +0x336
google.golang.org/grpc.getChainUnaryHandler.func1({0x1419348, 0xc00035b800}, {0x1162420, 0xc000248000})
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:1179 +0xb9
github.com/avos-io/iona/cwauth.Verify.func1({0x1419348, 0xc00035b800}, {0x1162420, 0xc000248000}, 0xc000201c00?, 0xc00027f480)
	/home/jon/source/iona/cwauth/verify.go:33 +0xc6
google.golang.org/grpc.chainUnaryInterceptors.func1({0x1419348, 0xc00035b800}, {0x1162420, 0xc000248000}, 0xc0004f1a20?, 0x1016e60?)
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:1170 +0x8f
github.com/avos-io/protorepo/gen/go/lindisfarne._Lindisfarne_ReportDynamicInfo_Handler({0x1193520?, 0xc0001dfc00}, {0x1419348, 0xc00035b800}, 0xc000100d20, 0xc000200500)
	/home/jon/source/iona/protorepo/gen/go/lindisfarne/lindisfarne_grpc.pb.go:314 +0x138
google.golang.org/grpc.(*Server).processUnaryRPC(0xc0002a8000, {0x141e420, 0xc0003fcf00}, 0xc0003bafc0, 0xc000391710, 0x1c175f8, 0x0)
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:1360 +0xe23
google.golang.org/grpc.(*Server).handleStream(0xc0002a8000, {0x141e420, 0xc0003fcf00}, 0xc0003bafc0, 0x0)
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:1737 +0xa2f
google.golang.org/grpc.(*Server).serveStreams.func1.1()
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:982 +0x98
created by google.golang.org/grpc.(*Server).serveStreams.func1
	/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go:980 +0x18c`,
		Result: &sentry.Event{
			Exception: []sentry.Exception{{
				Type:     "oh my god",
				ThreadID: "86",
				Mechanism: &sentry.Mechanism{
					Type: "panic",
					Data: make(map[string]interface{}),
				},
			}},
			Threads: []sentry.Thread{{
				ID: "86",
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{
						{
							Package:  "google.golang.org/grpc",
							Function: "Server.serveStreams.func1",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   980,
							InApp:    false,
						},
						{
							Package:  "google.golang.org/grpc",
							Function: "Server.serveStreams.func1.1",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   982,
							InApp:    false,
						},
						{
							Package:  "google.golang.org/grpc",
							Function: "Server.handleStream",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   1737,
							InApp:    false,
						},
						{
							Package:  "google.golang.org/grpc",
							Function: "Server.processUnaryRPC",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   1360,
							InApp:    false,
						},
						{
							Package:  "github.com/avos-io/protorepo/gen/go/lindisfarne",
							Function: "_Lindisfarne_ReportDynamicInfo_Handler",
							Filename: "/home/jon/source/iona/protorepo/gen/go/lindisfarne/lindisfarne_grpc.pb.go",
							Lineno:   314,
							InApp:    true,
						},
						{
							Package:  "google.golang.org/grpc",
							Function: "chainUnaryInterceptors.func1",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   1170,
							InApp:    false,
						},
						{
							Package:  "github.com/avos-io/iona/cwauth",
							Function: "Verify.func1",
							Filename: "/home/jon/source/iona/cwauth/verify.go",
							Lineno:   33,
							InApp:    true,
						},
						{
							Package:  "google.golang.org/grpc",
							Function: "getChainUnaryHandler.func1",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   1179,
							InApp:    false,
						},
						{
							Package:  "github.com/avos-io/iona/lindisfarne/internal/endpoints",
							Function: "Interceptor.Unary.func1",
							Filename: "/home/jon/source/iona/lindisfarne/internal/endpoints/interceptor.go",
							Lineno:   60,
							InApp:    true,
						},
						{
							Package:  "google.golang.org/grpc",
							Function: "getChainUnaryHandler.func1",
							Filename: "/home/jon/go/pkg/mod/google.golang.org/grpc@v1.57.0/server.go",
							Lineno:   1179,
							InApp:    false,
						},
						{
							Package:  "github.com/avos-io/iona/endpointauth",
							Function: "Interceptor.Unary.func1",
							Filename: "/home/jon/source/iona/endpointauth/interceptor.go",
							Lineno:   52,
							InApp:    true,
						},
						{
							Package:  "github.com/avos-io/protorepo/gen/go/lindisfarne",
							Function: "_Lindisfarne_ReportDynamicInfo_Handler.func1",
							Filename: "/home/jon/source/iona/protorepo/gen/go/lindisfarne/lindisfarne_grpc.pb.go",
							Lineno:   312,
							InApp:    true,
						},
						{
							Package:  "github.com/avos-io/iona/lindisfarne/internal/endpoints",
							Function: "Server.ReportDynamicInfo",
							Filename: "/home/jon/source/iona/lindisfarne/internal/endpoints/endpoints.go",
							Lineno:   868,
							InApp:    true,
						},
					},
				},
			}},
			Level: "fatal",
		},
	},
	"multiple goroutines": {
		Data: `panic: Something went wrong in packageA.foo()

goroutine 1 [running]:
github.com/user/packageA.foo()
	/path/to/packageA/foo.go:10
github.com/user/packageB.bar()
	/path/to/packageB/bar.go:15
main.main()
	/path/to/main.go:8

goroutine 2 [running]:
main.anotherFunction()
	/path/to/main.go:20
created by main.main
	/path/to/main.go:25`,
		Result: &sentry.Event{
			Exception: []sentry.Exception{{
				Type:     "Something went wrong in packageA.foo()",
				ThreadID: "1",
				Mechanism: &sentry.Mechanism{
					Type: "panic",
					Data: make(map[string]interface{}),
				},
			}},
			Threads: []sentry.Thread{
				{
					ID: "1",
					Stacktrace: &sentry.Stacktrace{
						Frames: []sentry.Frame{
							{
								Package:  "main",
								Function: "main",
								Filename: "/path/to/main.go",
								Lineno:   8,
								InApp:    true,
							},
							{
								Package:  "github.com/user/packageB",
								Function: "bar",
								Filename: "/path/to/packageB/bar.go",
								Lineno:   15,
								InApp:    true,
							},
							{
								Package:  "github.com/user/packageA",
								Function: "foo",
								Filename: "/path/to/packageA/foo.go",
								Lineno:   10,
								InApp:    true,
							},
						},
					},
				},
				{
					ID: "2",
					Stacktrace: &sentry.Stacktrace{
						Frames: []sentry.Frame{
							{
								Package:  "main",
								Function: "main",
								Filename: "/path/to/main.go",
								Lineno:   25,
								InApp:    true,
							},
							{
								Package:  "main",
								Function: "anotherFunction",
								Filename: "/path/to/main.go",
								Lineno:   20,
								InApp:    true,
							},
						},
					},
				},
			},
			Level: "fatal",
		},
	},
	"frames elided": {
		Data: `panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0xffffffff addr=0x0 pc=0x20314]

goroutine 1 [running]:
panic(0x112c00, 0x1040a038)
	/usr/local/go/src/runtime/panic.go:500 +0x720
main.aFunction()
	/tmp/sandbox675251439/main.go:23 +0x314
...additional frames elided...`,
		Result: &sentry.Event{
			Message: "invalid memory address or nil pointer dereference",
			Exception: []sentry.Exception{{
				Type:     "runtime error",
				Value:    "invalid memory address or nil pointer dereference",
				ThreadID: "1",
				Mechanism: &sentry.Mechanism{
					Type:        "signal",
					Data:        map[string]interface{}{"signal": "SIGSEGV", "code": "0xffffffff", "relevant_address": "0x0", "program_counter": "0x20314"},
					Description: "segmentation violation",
					Handled:     new(bool),
				},
			}},
			Threads: []sentry.Thread{{
				ID: "1",
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{
						{
							Package:  "main",
							Function: "aFunction",
							Filename: "/tmp/sandbox675251439/main.go",
							Lineno:   23,
							InApp:    true,
						},
						{
							Function: "panic",
							Filename: "/usr/local/go/src/runtime/panic.go",
							Lineno:   500,
							InApp:    false,
						},
					},
				},
			}},
			Level: "fatal",
		},
	},

	// Invalid input cases
	"empty": {
		Data:   "",
		Result: nil,
	},
	"no panic": {
		Data: `goroutine 1 [running]:
main.main()
	/tmp/sandbox675251439/main.go:23 +0x314`,
		Result: nil,
	},
	"no goroutines": {
		Data: `panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0xffffffff addr=0x0 pc=0x20314]`,
		Result: &sentry.Event{
			Message: "invalid memory address or nil pointer dereference",
			Exception: []sentry.Exception{{
				Type:  "runtime error",
				Value: "invalid memory address or nil pointer dereference",
				Mechanism: &sentry.Mechanism{
					Type:        "signal",
					Data:        map[string]interface{}{"signal": "SIGSEGV", "code": "0xffffffff", "relevant_address": "0x0", "program_counter": "0x20314"},
					Description: "segmentation violation",
					Handled:     new(bool),
				},
			}},
			Threads: nil,
			Level:   "fatal",
		},
	},
	"no frames": {
		Data: `panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0xffffffff addr=0x0 pc=0x20314]

goroutine 1 [running]:`,
		Result: &sentry.Event{
			Message: "invalid memory address or nil pointer dereference",
			Exception: []sentry.Exception{{
				Type:     "runtime error",
				Value:    "invalid memory address or nil pointer dereference",
				ThreadID: "1",
				Mechanism: &sentry.Mechanism{
					Type:        "signal",
					Data:        map[string]interface{}{"signal": "SIGSEGV", "code": "0xffffffff", "relevant_address": "0x0", "program_counter": "0x20314"},
					Description: "segmentation violation",
					Handled:     new(bool),
				},
			}},
			Threads: []sentry.Thread{{
				ID: "1",
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{},
				},
			}},
			Level: "fatal",
		},
	},
	"no file": {
		Data: `panic: oh nooooooooo

goroutine 1 [running]:
panic(0x112c00, 0x1040a038)`,
		Result: &sentry.Event{
			Exception: []sentry.Exception{{
				Type:     "oh nooooooooo",
				ThreadID: "1",
				Mechanism: &sentry.Mechanism{
					Type: "panic",
					Data: make(map[string]interface{}),
				},
			}},
			Threads: []sentry.Thread{{
				ID: "1",
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{
						{
							Function: "panic",
							Filename: "",
							Lineno:   0,
							InApp:    false,
						},
					},
				},
			},
			},
			Level: "fatal",
		},
	},
}

func compareEvents(t *testing.T, expected *sentry.Event, actual *sentry.Event) {
	is := assert.New(t)

	if expected == nil {
		is.Nil(actual)
		return
	}

	is.Equal(expected.Type, actual.Type, "Event Type")
	is.Equal(expected.Message, actual.Message, "Event Message")
	is.Equal(expected.Level, actual.Level, "Event Level")

	require.Equal(t, len(expected.Exception), len(actual.Exception), "Event Exceptions")
	for i := range actual.Exception {
		compareExceptions(t, &expected.Exception[i], &actual.Exception[i])
	}

	require.Equal(t, len(expected.Threads), len(actual.Threads), "Event Threads")
	for i := range actual.Threads {
		compareThreads(t, &expected.Threads[i], &actual.Threads[i])
	}
}

func compareExceptions(t *testing.T, expected *sentry.Exception, actual *sentry.Exception) {
	is := assert.New(t)

	require.NotNil(t, expected)
	require.NotNil(t, actual)

	is.Equal(expected.Type, actual.Type, "Exception Type")
	is.Equal(expected.Value, actual.Value, "Exception Value")
	is.Equal(expected.ThreadID, actual.ThreadID, "Exception ThreadID")
	is.Equal(expected.Stacktrace, actual.Stacktrace, "Exception Stacktrace")

	if actual.Mechanism != nil {
		compareMechanisms(t, expected.Mechanism, actual.Mechanism)
	}
}

func compareMechanisms(t *testing.T, expected *sentry.Mechanism, actual *sentry.Mechanism) {
	is := assert.New(t)

	require.NotNil(t, expected)
	require.NotNil(t, actual)

	is.Equal(expected.Type, actual.Type, "Mechanism Type")
	is.Equal(expected.Description, actual.Description, "Mechanism Description")
	is.Equal(expected.Data, actual.Data, "Mechanism Data")
	is.Equal(expected.Handled, actual.Handled, "Mechanism Handled")
}

func compareThreads(t *testing.T, expected *sentry.Thread, actual *sentry.Thread) {
	is := assert.New(t)

	require.NotNil(t, expected)
	require.NotNil(t, actual)

	is.Equal(expected.ID, actual.ID, "Thread ID")

	compareStacktrace(t, expected.Stacktrace, actual.Stacktrace)
}

func compareStacktrace(t *testing.T, expected *sentry.Stacktrace, actual *sentry.Stacktrace) {
	is := assert.New(t)

	require.NotNil(t, expected)
	require.NotNil(t, actual)

	is.Equal(expected.Frames, actual.Frames, "Stacktrace Frames")
}
