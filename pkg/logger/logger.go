package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

type contextKey string

const requestIDKey contextKey = "request_id"

type Handler struct {
	groups []string
	attrs  []slog.Attr

	opts Options

	mu  *sync.Mutex
	out io.Writer
}

// NewHandler creates a new Handler with the specified options. If opts is nil, uses [DefaultOptions].
func NewHandler(out io.Writer, opts *Options) *Handler {
	h := &Handler{out: out, mu: &sync.Mutex{}}
	if opts == nil {
		h.opts = *DefaultOptions
	} else {
		h.opts = *opts
	}
	return h
}

func (h *Handler) clone() *Handler {
	return &Handler{
		groups: h.groups,
		attrs:  h.attrs,
		opts:   h.opts,
		mu:     h.mu,
		out:    h.out,
	}
}

// Enabled implements slog.Handler.Enabled .
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle implements slog.Handler.Handle .
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	bf := getBuffer()
	bf.Reset()

	if !r.Time.IsZero() {
		fmt.Fprint(bf, color.New(color.Faint).Sprint(r.Time.Format(h.opts.TimeFormat)))
		fmt.Fprint(bf, " ")
	}

	if requestID, ok := RequestIDFromContext(ctx); ok {
		fmt.Fprint(bf, color.New(color.FgMagenta).Sprintf("%d ", requestID))
	}

	switch r.Level {
	case slog.LevelDebug:
		fmt.Fprint(bf, color.New(color.BgCyan, color.FgHiWhite).Sprint("DEBUG"))
	case slog.LevelInfo:
		fmt.Fprint(bf, color.New(color.BgGreen, color.FgHiWhite).Sprint("INFO "))
	case slog.LevelWarn:
		fmt.Fprint(bf, color.New(color.BgYellow, color.FgHiWhite).Sprint("WARN "))
	case slog.LevelError:
		fmt.Fprint(bf, color.New(color.BgRed, color.FgHiWhite).Sprint("ERROR"))
	}
	fmt.Fprint(bf, " ")

	if h.opts.SrcFileMode != Nop {
		if r.PC != 0 {
			f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()

			var filename string
			switch h.opts.SrcFileMode {
			case Nop:
				break
			case ShortFile:
				filename = filepath.Base(f.File)
			case LongFile:
				filename = f.File
			}
			lineStr := fmt.Sprintf(":%d", f.Line)
			formatted := fmt.Sprintf("%s ", filename+lineStr)
			if h.opts.SrcFileLength > 0 {
				maxFilenameLen := h.opts.SrcFileLength - len(lineStr) - 1
				if len(filename) > maxFilenameLen {
					filename = filename[:maxFilenameLen] // Truncate if too long
				}
				lenStr := strconv.Itoa(h.opts.SrcFileLength)
				formatted = fmt.Sprintf("%-"+lenStr+"s", filename+lineStr)
			}
			fmt.Fprint(bf, formatted)
		}
	}

	// we need the attributes here, as we can print a longer string if there are no attributes
	var attrs []slog.Attr
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	fmt.Fprint(bf, h.opts.MsgPrefix)
	formattedMessage := r.Message
	if h.opts.MsgLength > 0 && len(attrs) > 0 {
		if len(formattedMessage) > h.opts.MsgLength {
			formattedMessage = formattedMessage[:h.opts.MsgLength-1] + "…" // Truncate and add ellipsis if too long
		} else {
			// Pad with spaces if too short
			lenStr := strconv.Itoa(h.opts.MsgLength)
			formattedMessage = fmt.Sprintf("%-"+lenStr+"s", formattedMessage)
		}
	}
	if h.opts.MsgColor == nil {
		h.opts.MsgColor = color.New() // set to empty otherwise we have a null pointer
	}
	fmt.Fprintf(bf, "%s", h.opts.MsgColor.Sprint(formattedMessage))

	for _, a := range attrs {
		fmt.Fprint(bf, " ")
		for i, g := range h.groups {
			fmt.Fprint(bf, color.New(color.FgCyan).Sprint(g))
			if i != len(h.groups) {
				fmt.Fprint(bf, color.New(color.FgCyan).Sprint("."))
			}
		}

		if strings.Contains(a.Key, "err") {
			fmt.Fprint(bf, color.New(color.FgRed).Sprintf("%s=", a.Key)+a.Value.String())
		} else {
			fmt.Fprint(bf, color.New(color.FgCyan).Sprintf("%s=", a.Key)+a.Value.String())
		}
	}

	fmt.Fprint(bf, "\n")

	if h.opts.NoColor {
		stripANSI(bf)
	}

	h.mu.Lock()
	_, err := io.Copy(h.out, bf)
	h.mu.Unlock()

	freeBuffer(bf)

	return err
}

// WithGroup implements slog.Handler.WithGroup .
func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

// WithAttrs implements slog.Handler.WithAttrs .
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := h.clone()
	h2.attrs = append(h2.attrs, attrs...)
	return h2
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func getBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func freeBuffer(bf *bytes.Buffer) {
	bufPool.Put(bf)
}

func Prefix(prefix string, msg ...string) string {
	if len(msg) == 0 {
		return color.New(color.BgHiWhite, color.FgBlack).Sprint(prefix)
	}

	return color.New(color.BgHiWhite, color.FgBlack).Sprint(prefix) + " " + strings.Join(msg, " ")
}

type SourceFileMode int

const (
	// Nop does nothing.
	Nop SourceFileMode = iota

	// ShortFile produces only the filename (for example main.go:69).
	ShortFile

	// LongFile produces the full file path (for example /home/frajer/go/src/myapp/main.go:69).
	LongFile
)

// re is the regular expression used for removing ANSI colors.
var re = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// stripANSI removes ANSI escape sequences from the provided bytes.Buffer.
func stripANSI(bf *bytes.Buffer) {
	b := bf.Bytes()
	cleaned := re.ReplaceAll(b, nil)
	bf.Reset()
	bf.Write(cleaned)
}

var DefaultOptions = &Options{
	Level:         slog.LevelDebug,
	TimeFormat:    time.DateTime,
	SrcFileMode:   ShortFile,
	SrcFileLength: 0,
	MsgPrefix:     color.HiWhiteString("| "),
	MsgLength:     0,
	MsgColor:      color.New(),
	NoColor:       false,
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler

	// TimeFormat is the time format.
	TimeFormat string

	// SrcFileMode is the source file mode.
	SrcFileMode SourceFileMode

	// SrcFileLength to show fixed length filename to line up the log output, default 0 shows complete filename.
	SrcFileLength int

	// MsgPrefix to show prefix before message, default: white colored "| ".
	MsgPrefix string

	// MsgColor is the color of the message, default to empty.
	MsgColor *color.Color

	// MsgLength to show fixed length message to line up the log output, default 0 shows complete message.
	MsgLength int

	// NoColor disables color, default: false.
	NoColor bool
}

func ContextWithRequestID(ctx context.Context, requestID int64) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) (int64, bool) {
	correlationID, ok := ctx.Value(requestIDKey).(int64)
	return correlationID, ok
}
