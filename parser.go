package panicparse

import (
	"bufio"
	"bytes"
	"go/build"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog/log"
)

var (
	panicRegexp     = regexp.MustCompile(`^panic: (.*)$`)
	signalRegexp    = regexp.MustCompile(`^\[signal\s([^:]+):\s(.*)\]$`)
	goroutineRegexp = regexp.MustCompile(`^goroutine (\d+) \[([^,]+)(?:, (\d+) minutes)?(, locked to thread)?\]:$`)
	funcRegexp      = regexp.MustCompile(`^(created by )?(?:([^\/\(]*\/?[^\.\(]*)\.)?(?:\((\*)?([^\)]+)\))?\.?([^\(]+)(?:\(([^\)]*)\))?$`)
	fileRegexp      = regexp.MustCompile(`^\s*(.+):(\d+)\s*(.*)$`)

	framesElided = []byte("...additional frames elided...")
)

type state int

const (
	stateInit state = iota
	statePanic
	stateSignal
	stateStackFunc
	stateStackFile
)

type Event struct {
	Panic   *Panic
	Threads []*Goroutine
	Level   string
}

type Panic struct {
	Type        string
	Description string
	Synthetic   bool
	Signal      string
	SignalInfo  string
	Code        string
	Address     string
	PC          string
	ThreadId    string
}

type Goroutine struct {
	ID           string
	State        string
	Frames       []*Frame
	FramesElided bool
}

type Frame struct {
	RawFunc     string
	Package     string
	Receiver    string
	Pointer     bool
	Func        string
	File        string
	Line        int
	Arguments   []string
	StackOffset int64
}

func Parse(trace io.Reader) *sentry.Event {
	scanner := bufio.NewScanner(trace)

	state := stateInit

	panic := Panic{
		Type: "crash",
	}

	threads := []*Goroutine{}

	var goroutine *Goroutine
	var frame *Frame

	for scanner.Scan() {
		line := scanner.Bytes()

	restartSwitch:
		switch state {
		case stateInit:
			matches := panicRegexp.FindSubmatch(line)
			if matches == nil {
				continue
			}

			panic.Type = string(matches[1])
			state = statePanic

		case statePanic:
			matches := signalRegexp.FindSubmatch(line)
			if matches == nil {
				state = stateSignal
				continue
			}

			sigInfo := strings.Split(panic.Type, ": ")

			panic.Type = sigInfo[0]
			if len(sigInfo) > 1 {
				panic.Description = sigInfo[1]
			}
			panic.Synthetic = true

			signal := string(matches[1])
			extra := string(matches[2])

			extraInfo := strings.Split(extra, " ")
			panicInfo := []string{}

			for _, info := range extraInfo {
				if strings.Contains(info, "=") {
					parts := strings.Split(info, "=")
					switch parts[0] {
					case "code":
						panic.Code = parts[1]
					case "addr":
						panic.Address = parts[1]
					case "pc":
						panic.PC = parts[1]
					}
				} else {
					panicInfo = append(panicInfo, info)
				}
			}

			panic.Signal = signal
			panic.SignalInfo = strings.Join(panicInfo, " ")

		case stateSignal:
			matches := goroutineRegexp.FindSubmatch(line)
			if matches == nil {
				continue
			}

			id := string(matches[1])

			// I think the first thread we see is the one that panicked
			if panic.ThreadId == "" {
				panic.ThreadId = id
			}

			goroutine = &Goroutine{
				ID:    id,
				State: string(matches[2]),
			}
			threads = append(threads, goroutine)

			state = stateStackFunc

		case stateStackFunc:
			matches := funcRegexp.FindSubmatch(line)
			if matches == nil {
				if bytes.HasPrefix(line, framesElided) {
					goroutine.FramesElided = true
					continue
				}
				state = stateSignal
				goto restartSwitch
			}

			frame = &Frame{
				RawFunc:   string(matches[0]),
				Package:   string(matches[2]),
				Pointer:   len(matches[3]) > 0,
				Receiver:  string(matches[4]),
				Func:      string(matches[5]),
				Arguments: strings.Split(string(matches[5]), ", "),
			}
			goroutine.Frames = append(goroutine.Frames, frame)

			state = stateStackFile

		case stateStackFile:
			if frame == nil {
				log.Error().Msg("frame is nil")
				state = stateSignal
			}

			matches := fileRegexp.FindSubmatch(line)
			if matches == nil {
				state = stateSignal
				continue
			}

			line, err := strconv.Atoi(string(matches[2]))
			if err != nil {
				log.Err(err).Str("line", string(matches[2])).Msg("failed to parse line number")
				line = 0
			}

			offset, err := strconv.ParseInt(string(matches[3]), 0, 64)
			if err != nil {
				log.Err(err).Str("offset", string(matches[3])).Msg("failed to parse stack offset")
				offset = 0
			}

			frame.File = string(matches[1])
			frame.Line = line
			frame.StackOffset = offset

			state = stateStackFunc
		}
	}

	return eventToSentryEvent(&Event{
		Panic:   &panic,
		Threads: threads,
		Level:   "fatal",
	})
}

func eventToSentryEvent(e *Event) *sentry.Event {
	event := sentry.NewEvent()
	event.Message = e.Panic.Description
	event.Level = sentry.LevelFatal

	event.Exception = []sentry.Exception{
		*panicToSentryException(e.Panic),
	}

	event.Threads = goroutinesToSentryThreads(e.Threads)

	return event
}

func panicToSentryException(p *Panic) *sentry.Exception {
	mechanism := &sentry.Mechanism{
		Type: "panic",
		Data: make(map[string]interface{}),
	}

	if p.Signal != "" {
		handled := false

		mechanism.Type = "signal"
		mechanism.Data["signal"] = p.Signal
		mechanism.Data["code"] = p.Code
		mechanism.Description = p.SignalInfo
		mechanism.Handled = &handled
		if p.Address != "" {
			mechanism.Data["relevant_address"] = p.Address
		}
		if p.PC != "" {
			mechanism.Data["program_counter"] = p.PC
		}
	}

	exception := &sentry.Exception{
		Type:      p.Type,
		Value:     p.Description,
		ThreadID:  p.ThreadId,
		Mechanism: mechanism,
	}

	return exception
}

func goroutinesToSentryThreads(threads []*Goroutine) []sentry.Thread {
	sentryThreads := make([]sentry.Thread, len(threads))

	for i, thread := range threads {
		numFrames := len(thread.Frames)

		stacktrace := &sentry.Stacktrace{
			Frames: make([]sentry.Frame, numFrames),
		}

		// TODO: I don't understand how this is supposed to work yet
		// and there's no documentation for the field
		//if thread.FramesElided {
		//	stacktrace.FramesOmitted = []uint{1}
		//}

		for j, f := range thread.Frames {
			var fun string
			if f.Receiver != "" {
				fun = f.Receiver + "." + f.Func
			} else {
				fun = f.Func
			}

			inApp := !(strings.HasPrefix(f.File, build.Default.GOROOT) ||
				strings.Contains(f.File, "go/pkg/mod") ||
				strings.Contains(f.Package, "vendor") ||
				strings.Contains(f.Package, "third_party"))

			stacktrace.Frames[numFrames-j-1] = sentry.Frame{
				Package:  f.Package,
				Function: fun,
				Filename: f.File,
				Lineno:   f.Line,
				InApp:    inApp,
			}
		}

		sentryThreads[i] = sentry.Thread{
			ID:         thread.ID,
			Stacktrace: stacktrace,
		}
	}

	return sentryThreads
}
