package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func prometheusMiddleware() func(next http.Handler) http.Handler {
	durationHistorgram := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "request_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path", "method", "status_code"})
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lrw := &statusCodeKeepingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			route := mux.CurrentRoute(r)
			path, _ := route.GetPathTemplate()
			start := time.Now()

			next.ServeHTTP(lrw, r)

			duration := time.Since(start).Seconds()
			durationHistorgram.
				WithLabelValues(path, r.Method, strconv.Itoa(lrw.StatusCode())).
				Observe(duration)
		})
	}
}

// statusCodeKeepingResponseWriter is an http.ResponseWriter which stores the
// HTTP status code for later inspection.
type statusCodeKeepingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusCodeKeepingResponseWriter) StatusCode() int {
	return s.statusCode
}

func (s *statusCodeKeepingResponseWriter) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusCodeKeepingResponseWriter) Write(content []byte) (int, error) {
	return s.ResponseWriter.Write(content)
}
