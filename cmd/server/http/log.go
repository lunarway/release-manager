package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/lunarway/release-manager/internal/log"
)

type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *statusCodeResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// reqrespLogger returns an http.Handler that logs request and response
// details.
func reqrespLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &statusCodeResponseWriter{w, http.StatusOK}
		start := time.Now()
		h.ServeHTTP(lrw, r)
		if r.URL.Path == "/ping" {
			return
		}
		// request duration in miliseconds
		duration := time.Since(start).Nanoseconds() / 1e6
		statusCode := lrw.statusCode
		logger := log.With(
			"req", struct {
				URL     string            `json:"url,omitempty"`
				Method  string            `json:"method,omitempty"`
				Path    string            `json:"path,omitempty"`
				Headers map[string]string `json:"headers,omitempty"`
			}{
				URL:     r.URL.RequestURI(),
				Method:  r.Method,
				Path:    r.URL.Path,
				Headers: secureHeaders(flattenHeaders(r.Header)),
			},
			"res", struct {
				StatusCode int `json:"statusCode,omitempty"`
			}{
				StatusCode: statusCode,
			},
			"responseTime", duration,
		)
		if statusCode >= http.StatusInternalServerError {
			logger.Errorf("[%d] %s %s", statusCode, r.Method, r.URL.Path)
			return
		}
		logger.Infof("[%d] %s %s", statusCode, r.Method, r.URL.Path)
	})
}

// flattenHeaders flattens an http.Header map into a string map.
//
// Headers can contain multiple values so their are concatenated into a single
// string with , as separation.
func flattenHeaders(h http.Header) map[string]string {
	m := make(map[string]string)
	for key, values := range h {
		m[key] = strings.Join(values, ",")
	}
	return m
}

// secureHeaders copies header map h and removes sensitive information from the
// returned map.
func secureHeaders(h map[string]string) map[string]string {
	m := make(map[string]string)
	for key, value := range h {
		// crop contents of Bearer tokens to four characters
		if key != "Authorization" {
			m[key] = value
			continue
		}
		if strings.HasPrefix(value, "Bearer") {
			m[key] = cropBearer(value)
			continue
		}
		if len(value) > 4 {
			m[key] = value[:4]
			continue
		}
		m[key] = value
	}
	return m
}

// cropBearer crops tokens on the first four characters of the token
// Bearer 12345678 -> Bearer 1234
func cropBearer(value string) string {
	if len(value) > 11 {
		return value[:11]
	}
	return value
}
