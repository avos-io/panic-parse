package panicparse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"go/build"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

var (
	panicRegexp     = regexp.MustCompile(`^panic: (.*)$`)
	signalRegexp    = regexp.MustCompile(`^\[signal\s([^:]+):\s(.*)\]$`)
	goroutineRegexp = regexp.MustCompile(`^goroutine (\d+) \[([^,]+)(?:, (\d+) minutes)?(, locked to thread)?\]:$`)
	funcRegexp      = regexp.MustCompile(`^(created by )?([^\(]+)(?:\.\((\*)?([^\)]+)\))?\.([^\(]+)(?:\((.*)\))?$`)
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
	Id         uuid.UUID
	Panic      *Panic
	Threads    []*Goroutine
	Level      string
	ServerName string
	Release    string
	Tags       map[string]string
	Enviroment string
	Extra      map[string]interface{}
}

func (e *Event) MarshalJSON() ([]byte, error) {
	uuid := e.Id.String()
	id := strings.Replace(uuid, "-", "", -1)

	return json.Marshal(&struct {
		Id         string                 `json:"event_id"`
		Exception  []*Panic               `json:"exception"`
		Threads    []*Goroutine           `json:"threads"`
		Platform   string                 `json:"platform"`
		Level      string                 `json:"level"`
		ServerName string                 `json:"server_name,omitempty"`
		Release    string                 `json:"release,omitempty"`
		Tags       map[string]string      `json:"tags,omitempty"`
		Enviroment string                 `json:"environment,omitempty"`
		Extra      map[string]interface{} `json:"extra,omitempty"`
	}{
		Id:         id,
		Exception:  []*Panic{e.Panic},
		Threads:    e.Threads,
		Platform:   "go",
		Level:      e.Level,
		ServerName: e.ServerName,
		Release:    e.Release,
		Tags:       e.Tags,
		Enviroment: e.Enviroment,
		Extra:      e.Extra,
	})
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
	ThreadId    int
}

func (p *Panic) MarshalJSON() ([]byte, error) {
	type Signal struct {
		Name string `json:"name"`
		Code int64  `json:"code,omitempty"`
	}

	type Meta struct {
		Signal *Signal `json:"signal,omitempty"`
	}

	type Mechanism struct {
		Type        string                 `json:"type"`
		Description string                 `json:"description,omitempty"`
		Meta        *Meta                  `json:"meta,omitempty"`
		Data        map[string]interface{} `json:"data,omitempty"`
	}

	var synthetic *bool
	if p.Synthetic {
		synthetic = &p.Synthetic
	}

	mechanism := Mechanism{
		Type: "panic",
		Data: make(map[string]interface{}),
	}
	if p.Signal != "" {
		code, err := strconv.ParseInt(p.Code, 0, 32)
		if err != nil {
			code = 0
		}

		mechanism.Type = "signal"
		mechanism.Meta = &Meta{
			Signal: &Signal{
				Name: p.Signal,
				Code: code,
			},
		}
		mechanism.Description = p.SignalInfo
		if p.Address != "" {
			mechanism.Data["relevant_address"] = p.Address
		}
		if p.PC != "" {
			mechanism.Data["program_counter"] = p.PC
		}
	}

	return json.Marshal(&struct {
		Type      string    `json:"type"`
		Value     string    `json:"value"`
		Synthetic *bool     `json:"synthetic,omitempty"`
		Mechanism Mechanism `json:"mechanism"`
		ThreadId  int       `json:"thread_id,omitempty"`
	}{
		Type:      p.Type,
		Value:     p.Description,
		Synthetic: synthetic,
		Mechanism: mechanism,
		ThreadId:  p.ThreadId,
	})
}

type Goroutine struct {
	ID           int
	State        string
	Frames       []*Frame
	FramesElided bool
}

func (g *Goroutine) MarshalJSON() ([]byte, error) {
	type StackTrace struct {
		Frames []*Frame `json:"frames"`
	}

	return json.Marshal(&struct {
		ID         int        `json:"id"`
		State      string     `json:"state"`
		StackTrace StackTrace `json:"stacktrace"`
	}{
		ID:    g.ID,
		State: g.State,
		StackTrace: StackTrace{
			Frames: g.Frames,
		},
	})
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

func (f *Frame) MarshalJSON() ([]byte, error) {
	var fun string
	if f.Receiver != "" {
		fun = f.Receiver + "." + f.Func
	} else {
		fun = f.Func
	}

	inApp := !(strings.HasPrefix(f.File, build.Default.GOROOT) ||
		strings.Contains(f.Package, "vendor") ||
		strings.Contains(f.Package, "third_party"))

	return json.Marshal(&struct {
		Package string `json:"module"`
		Func    string `json:"function"`
		File    string `json:"filename"`
		Line    int    `json:"lineno"`
		RawFunc string `json:"raw_function"`
		InApp   bool   `json:"in_app"`
	}{
		Package: f.Package,
		Func:    fun,
		File:    f.File,
		Line:    f.Line,
		RawFunc: f.RawFunc,
		InApp:   inApp,
	})
}

func Parse(trace io.Reader) *Event {
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

			id, err := strconv.Atoi(string(matches[1]))
			if err != nil {
				log.Err(err).Str("id", string(matches[1])).Msg("failed to parse goroutine id")
				id = 0
			}

			// I think the first thread we see is the one that panicked
			if panic.ThreadId == 0 {
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

	// Sentry expects the frames to be ordered from oldest to newest
	for _, goroutine := range threads {
		slices.Reverse(goroutine.Frames)
	}

	return &Event{
		Id:      uuid.New(),
		Panic:   &panic,
		Threads: threads,
		Level:   "fatal",
		Tags:    make(map[string]string),
		Extra:   make(map[string]interface{}),
	}
}
