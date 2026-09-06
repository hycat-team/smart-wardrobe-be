package middleware

import (
	"bytes"
	"compress/gzip"
	"strings"

	"smart-wardrobe-be/internal/config"

	"github.com/gin-gonic/gin"
)

// GzipMinSizeBytes is the response size below which compression is skipped —
// compressing tiny payloads costs more than it saves (spec 019 FR-009).
const GzipMinSizeBytes = 1 << 10

var gzipExcludedPaths = []string{"/api/v1/ai/chat/sessions/*/messages/stream"}

// GzipMinSizeMiddleware compresses JSON responses with gzip only when the
// client accepts it and the response reaches GzipMinSizeBytes. The decision is
// deferred until enough bytes are written (or the first flush), so small
// payloads and SSE streams pass through uncompressed and unbuffered.
func GzipMinSizeMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Server.GzipEnabled || !clientAcceptsGzip(c) || isExcludedGzipPath(c.FullPath()) {
			c.Next()
			return
		}

		w := &gzipThresholdWriter{ResponseWriter: c.Writer, threshold: GzipMinSizeBytes}
		c.Writer = w
		c.Next()
		w.finalize()
	}
}

func clientAcceptsGzip(c *gin.Context) bool {
	return strings.Contains(c.GetHeader("Accept-Encoding"), "gzip")
}

func isExcludedGzipPath(path string) bool {
	for _, excluded := range gzipExcludedPaths {
		if matchGzipPattern(excluded, path) {
			return true
		}
	}
	return false
}

// matchGzipPattern supports the single `*` wildcard segment used by the SSE route exclusion.
func matchGzipPattern(pattern, path string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		return pattern == path
	}
	return strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1])
}

type gzipThresholdWriter struct {
	gin.ResponseWriter
	threshold  int
	buf        bytes.Buffer
	gz         *gzip.Writer
	decided    bool
	compressed bool
}

func (w *gzipThresholdWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipThresholdWriter) Write(b []byte) (int, error) {
	if !w.decided {
		w.buf.Write(b)
		if w.buf.Len() >= w.threshold {
			w.startCompression()
		}
		return len(b), nil
	}
	if w.compressed {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// Flush finalizes the size decision for streaming responses: the first flush
// below the threshold emits the buffered bytes raw and disables compression.
func (w *gzipThresholdWriter) Flush() {
	if !w.decided {
		w.decideRaw()
	}
	if w.compressed {
		_ = w.gz.Flush()
	}
	w.ResponseWriter.Flush()
}

func (w *gzipThresholdWriter) startCompression() {
	w.decided = true
	w.compressed = true
	w.Header().Del("Content-Length")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")
	w.gz = gzip.NewWriter(w.ResponseWriter)
	_, _ = w.gz.Write(w.buf.Bytes())
	w.buf.Reset()
}

func (w *gzipThresholdWriter) decideRaw() {
	w.decided = true
	w.compressed = false
	_, _ = w.ResponseWriter.Write(w.buf.Bytes())
	w.buf.Reset()
}

// finalize emits any buffered bytes and closes the gzip writer after the handler chain returns.
func (w *gzipThresholdWriter) finalize() {
	if !w.decided {
		w.decideRaw()
		return
	}
	if w.compressed {
		_ = w.gz.Close()
	}
}
