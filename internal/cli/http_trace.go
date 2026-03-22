package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

type verboseRoundTripper struct {
	base   http.RoundTripper
	logger *httpTraceLogger
}

func newVerboseRoundTripper(base http.RoundTripper, out io.Writer) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &verboseRoundTripper{
		base:   base,
		logger: newHTTPTraceLogger(out),
	}
}

func (t *verboseRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.logger.logRequest(req)
	started := time.Now()
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		t.logger.logError(err, time.Since(started))
		return nil, fmt.Errorf("round trip request: %w", err)
	}
	t.logger.logResponse(resp, time.Since(started))
	return resp, nil
}

type httpTraceLogger struct {
	out io.Writer
	dim *color.Color
	mu  sync.Mutex
}

func newHTTPTraceLogger(out io.Writer) *httpTraceLogger {
	useColor := supportsANSIColor(out)

	dim := color.New(color.Faint)
	if !useColor {
		dim.DisableColor()
	}

	return &httpTraceLogger{
		out: out,
		dim: dim,
	}
}

func (l *httpTraceLogger) logRequest(req *http.Request) {
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		l.writeSection("REQUEST", []byte(fmt.Sprintf("failed to dump request: %v\n", err)))
		return
	}
	l.writeSection("REQUEST", dump)
}

func (l *httpTraceLogger) logResponse(resp *http.Response, duration time.Duration) {
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		l.writeSection("RESPONSE", []byte(fmt.Sprintf("failed to dump response: %v\n", err)))
		return
	}
	l.writeSection(fmt.Sprintf("RESPONSE (%s)", duration.Round(time.Millisecond)), dump)
}

func (l *httpTraceLogger) logError(err error, duration time.Duration) {
	l.writeSection(
		fmt.Sprintf("RESPONSE ERROR (%s)", duration.Round(time.Millisecond)),
		[]byte(err.Error()+"\n"),
	)
}

func (l *httpTraceLogger) writeSection(title string, dump []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = l.dim.Fprintf(l.out, ">>> HTTP %s\n", title)
	_, _ = l.dim.Fprint(l.out, string(dump))
	if len(dump) == 0 || dump[len(dump)-1] != '\n' {
		_, _ = l.dim.Fprintln(l.out)
	}
	_, _ = l.dim.Fprintln(l.out)
}

func supportsANSIColor(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()
	return isatty.IsTerminal(fd)
}
