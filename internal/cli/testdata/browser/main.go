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
	mux.HandleFunc("/origin-frames", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Origin fixture</title><iframe id="blank"></iframe><iframe srcdoc="<h2>Srcdoc origin marker</h2>"></iframe><script>
blank.contentDocument.body.innerHTML = '<h2>Inherited origin marker</h2>';
const blobFrame = document.createElement('iframe');
blobFrame.src = URL.createObjectURL(new Blob(['<h2>Blob origin marker</h2>'], {type:'text/html'}));
document.body.append(blobFrame);
</script>`)
	})
	mux.HandleFunc("/opaque-frame", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Opaque fixture</title><script>Object.defineProperty(HTMLIFrameElement.prototype, 'contentDocument', {get() { return document; }});</script><iframe sandbox srcdoc="<h2>Opaque content must not leak</h2>"></iframe>`)
	})
	mux.HandleFunc("/closed-shadow", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Closed shadow fixture</title><div id="host"></div><p id="clickStatus">Click unchanged</p><p id="keyStatus">Key unchanged</p><p id="scrollStatus">Scroll unchanged</p><button id="cover">Cover closed action</button><script>
const outer = host.attachShadow({mode:'closed'});
outer.innerHTML='<div id="inner"></div>';
const root = outer.querySelector('#inner').attachShadow({mode:'closed'});
root.innerHTML='<button id="click">Closed action</button><input aria-label="Closed input"><div id="scroll" role="region" aria-label="Closed scroll" tabindex="0" style="height:80px;overflow:auto"><div style="height:500px">Closed scroll content</div></div>';
root.querySelector('#click').onclick=()=>clickStatus.textContent='Closed click received';
root.querySelector('input').onkeydown=e=>{if(e.key==='Enter')keyStatus.textContent='Closed key received'};
root.querySelector('#scroll').onscroll=()=>scrollStatus.textContent='Closed scroll received';
cover.onclick=()=>{const box=host.getBoundingClientRect();const overlay=document.createElement('div');Object.assign(overlay.style,{position:'fixed',left:box.left+'px',top:box.top+'px',width:box.width+'px',height:box.height+'px',zIndex:'9999',background:'white'});document.body.append(overlay)};
// Page-world overrides must not forge or interfere with native input proof.
Document.prototype.elementFromPoint=()=>null;
ShadowRoot.prototype.elementFromPoint=()=>null;
Node.prototype.getRootNode=()=>document;
</script>`)
	})
	mux.HandleFunc("/focus-redirect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Focus fixture</title><h1 id="status">Focus input untouched</h1><label>Redirecting input<input aria-label="Redirecting input" onfocus="document.getElementById('other').focus()"></label><label>Other input<input id="other" aria-label="Other input" onkeydown="document.getElementById('status').textContent='Unexpected keyboard input'" oninput="document.getElementById('status').textContent='Unexpected text input'"></label>`)
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
