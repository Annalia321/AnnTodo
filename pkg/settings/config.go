package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

// Config holds all runtime configuration read from conf/anntodo.ini.
// Every key can be overridden by an environment variable named
// ANNTODO_<SECTION>_<KEY>, e.g. ANNTODO_SERVER_PORT=8080.
type Config struct {
	ConfigFile string
	BaseDir    string // relative paths are resolved against this directory

	Server   ServerConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Log      LogConfig
	Session  SessionConfig
}

type ServerConfig struct {
	Host           string
	Port           int
	CertFile       string
	KeyFile        string
	TrustedProxies []string
}

type DatabaseConfig struct {
	Type     string // sqlite | mysql | postgres
	Path     string // sqlite file path
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type StorageConfig struct {
	DataDir string
}

type LogConfig struct {
	Level string // debug | info | warn | error
	File  string // empty means console only
}

type SessionConfig struct {
	RememberDays int // lifetime of "remember me" sessions
	DefaultHours int // sliding lifetime of normal sessions
}

// Addr returns host:port for the HTTP listener.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// TLSEnabled reports whether the server should serve HTTPS directly.
func (c *Config) TLSEnabled() bool {
	return c.Server.CertFile != "" && c.Server.KeyFile != ""
}

// DefaultConfigPath finds conf/anntodo.ini next to the executable, falling back
// to the current working directory.
func DefaultConfigPath() string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "conf", "anntodo.ini")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	p, _ := filepath.Abs(filepath.Join("conf", "anntodo.ini"))
	return p
}

// Load reads the config file (if it exists) and applies environment overrides.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	file := ini.Empty()
	if _, err := os.Stat(abs); err == nil {
		file, err = ini.Load(abs)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", abs, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	r := reader{file: file}
	cfg := &Config{ConfigFile: abs}

	// conf/anntodo.ini -> base dir is the parent of "conf".
	dir := filepath.Dir(abs)
	if strings.EqualFold(filepath.Base(dir), "conf") {
		dir = filepath.Dir(dir)
	}
	cfg.BaseDir = dir

	cfg.Server.Host = r.str("server", "host", "0.0.0.0")
	cfg.Server.Port = r.int("server", "port", 8080)
	cfg.Server.CertFile = cfg.resolve(r.str("server", "cert_file", ""))
	cfg.Server.KeyFile = cfg.resolve(r.str("server", "key_file", ""))
	cfg.Server.TrustedProxies = splitList(r.str("server", "trusted_proxies", "127.0.0.1,::1"))

	cfg.Storage.DataDir = cfg.resolve(r.str("storage", "data_dir", "data"))

	cfg.Database.Type = strings.ToLower(r.str("database", "type", "sqlite"))
	dbPath := r.str("database", "path", "")
	if dbPath == "" {
		cfg.Database.Path = filepath.Join(cfg.Storage.DataDir, "anntodo.db")
	} else {
		cfg.Database.Path = cfg.resolve(dbPath)
	}
	cfg.Database.Host = r.str("database", "host", "127.0.0.1")
	cfg.Database.Port = r.int("database", "port", 0)
	cfg.Database.Name = r.str("database", "name", "anntodo")
	cfg.Database.User = r.str("database", "user", "")
	cfg.Database.Password = r.str("database", "password", "")
	cfg.Database.SSLMode = r.str("database", "ssl_mode", "disable")

	cfg.Log.Level = strings.ToLower(r.str("log", "level", "info"))
	cfg.Log.File = cfg.resolve(r.str("log", "file", ""))

	cfg.Session.RememberDays = r.int("session", "remember_days", 30)
	cfg.Session.DefaultHours = r.int("session", "default_hours", 24)

	switch cfg.Database.Type {
	case "sqlite", "mysql", "postgres":
	default:
		return nil, fmt.Errorf("unsupported database type %q", cfg.Database.Type)
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid server port %d", cfg.Server.Port)
	}
	if cfg.Session.RememberDays <= 0 {
		cfg.Session.RememberDays = 30
	}
	if cfg.Session.DefaultHours <= 0 {
		cfg.Session.DefaultHours = 24
	}
	return cfg, nil
}

func (c *Config) resolve(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.BaseDir, p)
}

type reader struct{ file *ini.File }

func envName(section, key string) string {
	return "ANNTODO_" + strings.ToUpper(section) + "_" + strings.ToUpper(key)
}

func (r reader) str(section, key, def string) string {
	if v, ok := os.LookupEnv(envName(section, key)); ok {
		return strings.TrimSpace(v)
	}
	if r.file.Section(section).HasKey(key) {
		return strings.TrimSpace(r.file.Section(section).Key(key).String())
	}
	return def
}

func (r reader) int(section, key string, def int) int {
	s := r.str(section, key, "")
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
