package httplog

import (
	"encoding/base32"
	"encoding/binary"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type contextKey string

// TraceIDKey is key used to add a trace.id value to the context of HTTP
// requests. The value will be logged by LoggingRoundTripper.
var TraceIDKey = contextKey("trace.id")

// NewLoggingRoundTripper returns a new RoundTripper that logs requests and
// responses to the provided logger.
func NewLoggingRoundTripper(next http.RoundTripper, logger *zap.Logger) *LoggingRoundTripper {
	return &LoggingRoundTripper{
		transport:   next,
		logger:      logger,
		txBaseID:    newID(),
		txIDCounter: atomic.NewUint64(0),
	}
}

type LoggingRoundTripper struct {
	transport   http.RoundTripper
	logger      *zap.Logger
	txBaseID    string         // Random value to make transaction IDs unique.
	txIDCounter *atomic.Uint64 // Transaction ID counter that is incremented for each request.
}

// nextTxID returns the next transaction.id value. It increments the internal
// request counter.
func (rt *LoggingRoundTripper) nextTxID() string {
	count := rt.txIDCounter.Inc()
	return rt.txBaseID + "-" + strconv.FormatUint(count, 10)
}

func (rt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a child logger for this request.
	log := rt.logger.With(
		zap.String("transaction.id", rt.nextTxID()),
	)

	if v := req.Context().Value(TraceIDKey); v != nil {
		if traceID, ok := v.(string); ok {
			log = log.With(zap.String("trace.id", traceID))
		}
	}

	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		log.Debug("HTTP request",
			zap.String("url.original", req.URL.String()),
			zap.String("url.scheme", req.URL.Scheme),
			zap.String("url.path", req.URL.Path),
			zap.String("url.domain", req.URL.Hostname()),
			zap.String("url.port", req.URL.Port()),
			zap.String("url.query", req.URL.RawQuery),
			zap.String("http.request.method", req.Method),
			zap.ByteString("http.request.body.content", dump),
		)
	}

	resp, err := rt.transport.RoundTrip(req)
	if resp != nil {
		if dump, err := httputil.DumpResponse(resp, true); err == nil {
			log.Debug("HTTP response",
				zap.Int("http.response.status_code", resp.StatusCode),
				zap.ByteString("http.response.body.content", dump),
			)
		}
	} else {
		log.Debug("HTTP response error", zap.Error(err))
	}

	return resp, err
}

// newID returns an ID derived from the current time.
func newID() string {
	var data [8]byte
	binary.LittleEndian.PutUint64(data[:], uint64(time.Now().UnixNano()))
	return base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(data[:])
}
