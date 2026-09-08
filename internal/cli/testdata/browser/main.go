// Browser integration fixture: a lease-owned loopback HTTP backend.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync/atomic"
)

//go:embed page.html
var files embed.FS

func main() {
	port := flag.Int("port", 0, "loopback port")
	flag.Parse()
	var clicks atomic.Int64
	var lastText atomic.Value
	lastText.Store("")
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("/click", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
		if err != nil {
			http.Error(w, "invalid body", 400)
			return
		}
		lastText.Store(string(body))
		n := clicks.Add(1)
		fmt.Printf("browser-fixture click=%d\n", n)
		fmt.Fprint(w, n)
	})
	mux.HandleFunc("/last", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, lastText.Load()) })
	mux.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, clicks.Load()) })
	mux.HandleFunc("/tick", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "fixture-secret=never-record")
		fmt.Fprint(w, "tick")
	})
	mux.HandleFunc("/frame", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h2>Iframe heading</h2><button>Iframe action</button>")
	})
	mux.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<title>Next page</title><h1>Navigation complete</h1>")
	})
	mux.Handle("/", http.FileServer(http.FS(files)))
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		panic(err)
	}
	fmt.Printf("browser-fixture listening pid=%d port=%d\n", os.Getpid(), listener.Addr().(*net.TCPAddr).Port)
	if err := http.Serve(listener, mux); err != nil {
		panic(err)
	}
}
