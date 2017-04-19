package resources

import "net/http"

// MethodNotAllowed returns 405
func MethodNotAllowed() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(405)
		w.Write([]byte("Method not allowed"))
	}
}
