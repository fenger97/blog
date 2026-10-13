package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"blog/configs"
	"blog/internal/content"
	"blog/internal/handlers"
	"blog/internal/web"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = ""
)

func main() {
	cfg := configs.LoadConfig()
	siteURL, err := url.Parse(cfg.SiteURL)
	if err != nil || siteURL.Host == "" || (siteURL.Scheme != "http" && siteURL.Scheme != "https") || siteURL.User != nil || siteURL.RawQuery != "" || siteURL.Fragment != "" || strings.Trim(siteURL.Path, "/") != "" {
		log.Fatal("SITE_URL must be an absolute HTTP(S) site origin")
	}
	var store handlers.PostReader
	switch cfg.ContentMode {
	case "local":
		store = content.NewLocalStore(cfg.ContentDir)
	case "github":
		store = content.NewGitHubStore(content.Config{IndexURL: cfg.ContentIndexURL, BaseURL: cfg.ContentBaseURL, CacheTTL: cfg.ContentCacheTTL, AuthToken: cfg.GitHubToken, CacheDir: cfg.ContentCacheDir})
	default:
		log.Fatal("CONTENT_MODE must be github or local")
	}
	if commit == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
				}
				if setting.Key == "vcs.modified" && setting.Value == "true" {
					version += "+modified"
				}
			}
		}
	}
	router := web.New(store, web.Config{SiteURL: cfg.SiteURL, Version: version, Commit: commit, BuildTime: buildTime, ContentMode: cfg.ContentMode, ContentDir: cfg.ContentDir, ContentBaseURL: cfg.ContentBaseURL})
	server := &http.Server{Addr: cfg.ServerAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Shutdown: %v", err)
		}
	}()
	log.Printf("Server starting on %s (content=%s, version=%s, commit=%s)", cfg.ServerAddr, cfg.ContentMode, version, commit)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-shutdownDone
}

func newRouter(reader handlers.PostReader) http.Handler {
	return web.New(reader, web.Config{})
}
