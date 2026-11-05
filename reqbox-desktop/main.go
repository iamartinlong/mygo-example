// Command reqbox-desktop is the mygo desktop shell for Reqbox: the Reqbox
// web UI (embedded dist) served from a loopback HTTP server and shown in a
// native window. The store and runner are the unmodified Reqbox packages.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/egoist/mygo"

	"reqbox-desktop/internal/server"
	"reqbox-desktop/internal/store"
)

// web/dist is copied from the upstream Reqbox build (pnpm build in
// D:\opencode\Reqbox\web), committed so `go build` is self-contained.
//
//go:embed all:web/dist
var dist embed.FS

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	if err := run(); err != nil {
		slog.Error("reqbox-desktop", "error", err)
		os.Exit(1)
	}
}

func run() error {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(log)

	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return fmt.Errorf("user data dir: %w", err)
	}
	st, err := store.Open(dir + string(os.PathSeparator) + "reqbox.db")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()
	if st.PermissionWarning != nil {
		log.Warn("could not restrict database permissions", "error", st.PermissionWarning)
	}
	log.Info("using database", "path", dir)

	assets := fs.FS(dist)
	if _, err := fs.Stat(dist, "web/dist/index.html"); err != nil {
		return fmt.Errorf("web build missing: %w", err)
	}

	// Loopback, an ephemeral port: the UI is the only intended client of the
	// API; the mock gateway gets its stable endpoint in a later batch.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	handler := server.New(st, log, server.Config{Addr: ln.Addr().String()}, assets)
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil {
			log.Error("server stopped", "error", err)
		}
	}()
	defer srv.Close()
	baseURL := "http://" + ln.Addr().String()
	log.Info("reqbox-desktop started", "url", baseURL)

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     "Reqbox",
			URL:       baseURL + "/",
			Width:     1280,
			Height:    820,
			MinWidth:  960,
			MinHeight: 600,
		})
	})
	mygo.App.Run()
	return nil
}
