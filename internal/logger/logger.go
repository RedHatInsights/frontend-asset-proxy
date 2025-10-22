package logger

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/smithy-go/logging"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

type StructuredLogger struct {
	Logger   *logrus.Logger
	LogLevel logrus.Level
}

type LogEntry struct {
	*StructuredLogger
	request  *http.Request
	buf      *bytes.Buffer
	useColor bool
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &LogEntry{
		StructuredLogger: l,
		request:          r,
		buf:              &bytes.Buffer{},
		useColor:         false,
	}

	reqID := middleware.GetReqID(r.Context())
	if reqID != "" {
		fmt.Fprintf(entry.buf, "[%s] ", reqID)
	}

	fmt.Fprintf(entry.buf, "\"")
	fmt.Fprintf(entry.buf, "%s ", r.Method)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fmt.Fprintf(entry.buf, "%s://%s%s %s\" ", scheme, r.Host, r.RequestURI, r.Proto)

	entry.buf.WriteString("from ")
	entry.buf.WriteString(r.RemoteAddr)
	entry.buf.WriteString(" - ")

	return entry
}

func (l *LogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	// Do nothing if status code is 200/201/eg and the log level is above Warn (3)
	if (l.LogLevel <= logrus.WarnLevel) && (status < 400) {
		return
	}

	fmt.Fprintf(l.buf, "%03d", status)
	fmt.Fprintf(l.buf, " %dB", bytes)

	l.buf.WriteString(" in ")
	if elapsed < 500*time.Millisecond {
		fmt.Fprintf(l.buf, "%s", elapsed)
	} else if elapsed < 5*time.Second {
		fmt.Fprintf(l.buf, "%s", elapsed)
	} else {
		fmt.Fprintf(l.buf, "%s", elapsed)
	}

	l.Logger.Print(l.buf.String())
}

func (l *LogEntry) Panic(v interface{}, stack []byte) {
	middleware.PrintPrettyStack(v)
}

func NewLogger(lvl string, logger *logrus.Logger) *StructuredLogger {
	logLevel, err := logrus.ParseLevel(lvl)
	if err != nil {
		logLevel = logrus.ErrorLevel
	}
	logger.SetLevel(logLevel)
	return &StructuredLogger{
		Logger:   logger,
		LogLevel: logLevel,
	}
}

// ContextAwareLogger implements smithy logging.Logger and logging.ContextLogger.
// It enriches AWS SDK logs with chi's request ID when available.
type ContextAwareLogger struct{ Base *logrus.Logger }

type requestLoggerWithID struct {
	Base  *logrus.Logger
	ReqID string
}

func (l ContextAwareLogger) WithContext(ctx context.Context) logging.Logger {
	rid := middleware.GetReqID(ctx)
	return requestLoggerWithID{Base: l.Base, ReqID: rid}
}

// Fallback when no context is provided by the SDK
func (l ContextAwareLogger) Logf(class logging.Classification, format string, v ...interface{}) {
	entry := l.Base.WithFields(logrus.Fields{
		"process": "s3client",
	})
	logWith(entry, "", class, format, v...)
}

func (l requestLoggerWithID) Logf(class logging.Classification, format string, v ...interface{}) {
	entry := l.Base.WithFields(logrus.Fields{
		"process": "s3client",
	})
	if l.ReqID != "" {
		entry = entry.WithField("request_id", l.ReqID)
	}
	logWith(entry, l.ReqID, class, format, v...)
}

// map smithy classification to logrus level
func levelFor(class logging.Classification) logrus.Level {
	if class == logging.Warn {
		return logrus.WarnLevel
	}
	return logrus.DebugLevel
}

// logWith prefixes the message with the request ID (if any) and logs at the mapped level
func logWith(entry *logrus.Entry, reqID string, class logging.Classification, format string, v ...interface{}) {
	if reqID != "" {
		format = fmt.Sprintf("[%s] %s", reqID, format)
	}
	entry.Logf(levelFor(class), format, v...)
}
