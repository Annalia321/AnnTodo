package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata" // timezone database for hosts without one (Windows)

	"github.com/kardianos/service"

	"anntodo/pkg/api"
	"anntodo/pkg/core"
	mw "anntodo/pkg/middleware"
	"anntodo/pkg/settings"
	"anntodo/pkg/storage"
	"anntodo/pkg/version"
	"anntodo/web"
)

const usage = `AnnTodo %s - self-hosted todo manager

Usage:
  anntodo [-c conf/anntodo.ini] [run]       start the web server
  anntodo [-c conf/anntodo.ini] service install|uninstall|start|stop|restart|status
  anntodo version

`

func main() {
	flag.Usage = func() { fmt.Fprintf(os.Stderr, usage, version.Version); flag.PrintDefaults() }
	configPath := flag.String("c", "", "path to the config file (default: conf/anntodo.ini next to the executable)")
	flag.Parse()

	args := flag.Args()
	cmd := "run"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "version":
		fmt.Println("AnnTodo", version.Version)
	case "run":
		if err := runService(*configPath, ""); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "service":
		if len(args) < 2 {
			flag.Usage()
			os.Exit(2)
		}
		if err := runService(*configPath, args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}
}

type program struct {
	cfg    *settings.Config
	srv    *http.Server
	cancel context.CancelFunc
	done   chan struct{}
}

func runService(configPath, action string) error {
	if configPath == "" {
		configPath = settings.DefaultConfigPath()
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}
	svcConfig := &service.Config{
		Name:        "AnnTodo",
		DisplayName: "AnnTodo",
		Description: "AnnTodo self-hosted todo manager",
		Arguments:   []string{"-c", absConfig, "run"},
	}
	prg := &program{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}
	switch action {
	case "":
		cfg, err := settings.Load(absConfig)
		if err != nil {
			return err
		}
		prg.cfg = cfg
		return svc.Run()
	case "status":
		st, err := svc.Status()
		if err != nil {
			return err
		}
		fmt.Println(map[service.Status]string{service.StatusRunning: "running", service.StatusStopped: "stopped"}[st])
		return nil
	case "install", "uninstall", "start", "stop", "restart":
		if err := service.Control(svc, action); err != nil {
			return err
		}
		fmt.Printf("service %s: ok\n", action)
		return nil
	default:
		return fmt.Errorf("unknown service action %q", action)
	}
}

func (p *program) Start(s service.Service) error {
	if err := setupLogging(p.cfg); err != nil {
		return err
	}
	db, err := storage.Open(p.cfg)
	if err != nil {
		return err
	}
	app := core.New(db, p.cfg)
	server := &api.Server{App: app, Limiter: mw.NewLoginLimiter(), Web: web.FS()}

	p.srv = &http.Server{Addr: p.cfg.Addr(), Handler: server.Router(), ReadHeaderTimeout: 10 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})

	go housekeeping(ctx, app)
	go func() {
		defer close(p.done)
		scheme := "http"
		var err error
		slog.Info("AnnTodo starting", "version", version.Version, "addr", p.cfg.Addr(), "config", p.cfg.ConfigFile, "database", p.cfg.Database.Type)
		if p.cfg.TLSEnabled() {
			scheme = "https"
			slog.Info("listening", "url", scheme+"://"+displayAddr(p.cfg))
			err = p.srv.ListenAndServeTLS(p.cfg.Server.CertFile, p.cfg.Server.KeyFile)
		} else {
			slog.Info("listening", "url", scheme+"://"+displayAddr(p.cfg))
			err = p.srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "err", err)
			os.Exit(1)
		}
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = p.srv.Shutdown(ctx)
		<-p.done
	}
	slog.Info("AnnTodo stopped")
	return nil
}

func displayAddr(cfg *settings.Config) string {
	host := cfg.Server.Host
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, cfg.Server.Port)
}

// housekeeping removes expired sessions and trash older than 30 days.
func housekeeping(ctx context.Context, app *core.App) {
	run := func() {
		if err := app.PurgeExpiredSessions(); err != nil {
			slog.Warn("purge sessions", "err", err)
		}
		if err := app.PurgeOldTrash(); err != nil {
			slog.Warn("purge trash", "err", err)
		}
	}
	run()
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

func setupLogging(cfg *settings.Config) error {
	var lvl slog.Level
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	var w io.Writer = os.Stdout
	if cfg.Log.File != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.Log.File), 0o750); err != nil {
			return err
		}
		f, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
		if err != nil {
			return err
		}
		if service.Interactive() {
			w = io.MultiWriter(os.Stdout, f)
		} else {
			w = f
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl})))
	return nil
}
