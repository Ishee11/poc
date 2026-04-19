package http

import "net/http"

type observedResponseWriter struct {
	http.ResponseWriter
	status    int
	bytes     int
	errorCode string
}

func newObservedResponseWriter(w http.ResponseWriter) *observedResponseWriter {
	return &observedResponseWriter{ResponseWriter: w}
}

func (w *observedResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *observedResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

func (w *observedResponseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *observedResponseWriter) Bytes() int {
	return w.bytes
}

func (w *observedResponseWriter) SetErrorCode(code string) {
	w.errorCode = code
}

func (w *observedResponseWriter) ErrorCode() string {
	return w.errorCode
}

func (w *observedResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type errorCodeWriter interface {
	SetErrorCode(code string)
}

func setResponseErrorCode(w http.ResponseWriter, code string) {
	if ew, ok := w.(errorCodeWriter); ok {
		ew.SetErrorCode(code)
	}
}
