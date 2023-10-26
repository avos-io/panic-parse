package main_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/avos-io/gocrash/panicparse"
	"github.com/stretchr/testify/require"
)

var sentry = panicparse.FromDNS("your dsn here")

func TestPanicParse(t *testing.T) {
	for _, data := range testData {
		event := panicparse.Parse(strings.NewReader(data))
		event.Extra["panic"] = data

		json, _ := json.MarshalIndent(event, "", "  ")
		fmt.Printf("panic report: %v\n", string(json))

		_, err := sentry.SendCrashReport(event)
		require.NoError(t, err)
	}
}

var testData = []string{
	`panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0xffffffff addr=0x0 pc=0x20314]

goroutine 1 [running]:
panic(0x112c00, 0x1040a038)
	/usr/local/go/src/runtime/panic.go:500 +0x720
main.main()
	/tmp/sandbox675251439/main.go:23 +0x314`,
	`panic: oh my god

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
}
