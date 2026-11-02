package core

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"anntodo/pkg/settings"
)

// App bundles the dependencies shared by all business logic.
type App struct {
	DB  *gorm.DB
	Cfg *settings.Config
}

func New(db *gorm.DB, cfg *settings.Config) *App {
	return &App{DB: db, Cfg: cfg}
}

func now() time.Time { return time.Now().UTC() }

var (
	usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_.\-]{3,32}$`)
	colorRe    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	dateRe     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe     = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
)

func normalizeUsername(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !usernameRe.MatchString(s) {
		return "", ErrInvalidUsername
	}
	return s, nil
}

func validatePassword(p string) error {
	if utf8.RuneCountInString(p) < 8 || len(p) > 72 {
		return ErrInvalidPassword
	}
	return nil
}

// normalizeEmail returns nil for an empty email.
func normalizeEmail(s string) (*string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil, nil
	}
	if len(s) > 255 {
		return nil, ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return nil, ErrInvalidEmail
	}
	return &s, nil
}

func cleanText(s string, max int, field string, required bool) (string, error) {
	s = strings.TrimSpace(s)
	if required && s == "" {
		return "", BadRequest(field + " is required")
	}
	if utf8.RuneCountInString(s) > max {
		return "", BadRequest(field + " is too long")
	}
	return s, nil
}

func validColor(c string) bool { return c == "" || colorRe.MatchString(c) }

func validDate(d string) bool {
	if !dateRe.MatchString(d) {
		return false
	}
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

func validTime(t string) bool { return timeRe.MatchString(t) }
