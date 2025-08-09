package main

// overlay-hub exposes register and heartbeat endpoints and a QUIC relay stub.
//
// Example:
//   go run ./cmd/overlay-hub/main.go -addr :8090

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
)

var jwks *keyfunc.JWKS

func main() {
	addr := flag.String("addr", ":8090", "HTTP bind address")
	jwksURL := flag.String("jwks-url", os.Getenv("JWKS_URL"), "JWKS endpoint URL")
	flag.Parse()

	if *jwksURL == "" {
		log.Fatal("JWKS_URL is required")
	}
	var err error
	jwks, err = keyfunc.Get(*jwksURL, keyfunc.Options{RefreshInterval: time.Minute})
	if err != nil {
		log.Fatalf("load jwks: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/register", okHandler)
	mux.HandleFunc("/v1/heartbeat", okHandler)
	mux.HandleFunc("/v1/negotiate", okHandler)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Printf("overlay-hub listening on %s", *addr)
	log.Fatal(srv.ListenAndServe())
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	if jwks == nil {
		http.Error(w, "jwks not loaded", http.StatusInternalServerError)
		return
	}
	if _, err := jwt.Parse(token, jwks.Keyfunc); err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}
