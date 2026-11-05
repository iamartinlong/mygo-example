package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"reqbox-desktop/pkg/runner"
	"reqbox-desktop/pkg/store"
)

// startMockGateway runs the published-mock endpoint on a loopback port,
// preferring the upstream default 5178 so URLs survive a restart. The
// handler mirrors pkg/server's handleMock, minus token/UI concerns.
func (a *app) startMockGateway() {
	ln, err := net.Listen("tcp", "127.0.0.1:5178")
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Printf("mock gateway: %v", err)
			return
		}
		log.Printf("mock gateway: default port busy, using %s", ln.Addr())
	}
	a.mockAddr = ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/mock/", func(w http.ResponseWriter, r *http.Request) {
		a.serveMock(w, r)
	})
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil {
			log.Printf("mock gateway stopped: %v", err)
		}
	}()
}

func (a *app) serveMock(w http.ResponseWriter, r *http.Request) {
	slug := trimSlug(strings.TrimPrefix(r.URL.Path, "/mock"))
	if slug == "" {
		writeMockErr(w, http.StatusNotFound, "unknown mock")
		return
	}
	if a.st == nil {
		writeMockErr(w, http.StatusServiceUnavailable, "store unavailable")
		return
	}
	saved, err := a.st.GetRequestBySlug(slug)
	if err != nil {
		writeMockErr(w, http.StatusNotFound, "unknown mock")
		return
	}

	opts := runner.DefaultOpts()
	opts.Timeout = 60 * time.Second
	vars := map[string]string{}

	// Query parameters override variables, but may not move the call to a
	// host the owner did not configure - such callers lose the stored
	// credentials, exactly as the web gateway behaves.
	trusted := hostOfRequest(saved, vars)
	for key := range r.URL.Query() {
		vars[key] = r.URL.Query().Get(key)
	}
	if hostOfRequest(saved, vars) != trusted {
		saved.AuthType = "none"
		saved.Auth = store.Auth{}
		saved.Headers = nil
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		saved.Method = r.Method
	}

	res, runErr := a.exec.Run(r.Context(), saved, vars, opts)
	if runErr != nil {
		writeMockErr(w, http.StatusBadGateway, runErr.Error())
		return
	}
	ct := res.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(res.Status)
	_, _ = w.Write(res.Body)
}

func writeMockErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

func hostOfRequest(req store.Request, vars map[string]string) string {
	raw, err := runner.BuildURL(req.URL, req.Params, vars)
	if err != nil {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}

func trimSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
			b.WriteRune(c)
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}
