package models

import "time"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type User struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;not null;uniqueIndex" json:"username"`
	Email        *string    `gorm:"size:255;uniqueIndex" json:"email"`
	Nickname     string     `gorm:"size:64;not null;default:''" json:"nickname"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Role         string     `gorm:"size:16;not null;default:user" json:"role"`
	Status       string     `gorm:"size:16;not null;default:active" json:"status"`
	Avatar       string     `gorm:"size:255;not null;default:''" json:"-"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Card struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	OwnerID   uint64     `gorm:"not null;index" json:"owner_id"`
	Name      string     `gorm:"size:100;not null" json:"name"`
	Color     string     `gorm:"size:16;not null;default:''" json:"color"`
	SortOrder int64      `gorm:"not null;default:0" json:"sort_order"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Item struct {
	ID                uint64     `gorm:"primaryKey" json:"id"`
	CardID            uint64     `gorm:"not null;index" json:"card_id"`
	OwnerID           uint64     `gorm:"not null;index" json:"owner_id"`
	Title             string     `gorm:"size:500;not null" json:"title"`
	Note              string     `gorm:"type:text;not null;default:''" json:"note"`
	DueDate           *string    `gorm:"size:10;index" json:"due_date"` // YYYY-MM-DD; UTC date when DueTime is set
	DueTime           *string    `gorm:"size:5" json:"due_time"`        // HH:MM in UTC; nil means all day
	Important         bool       `gorm:"not null;default:false" json:"important"`
	Urgent            bool       `gorm:"not null;default:false" json:"urgent"`
	Completed         bool       `gorm:"not null;default:false" json:"completed"`
	CompletedAt       *time.Time `gorm:"index" json:"completed_at"`
	SortOrder         int64      `gorm:"not null;default:0" json:"sort_order"`
	QuadrantSortOrder int64      `gorm:"not null;default:0" json:"quadrant_sort_order"`
	DeletedAt         *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	DeletedWithCard   bool       `gorm:"not null;default:false" json:"-"` // deleted together with its card
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type UserSetting struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	Settings  string `gorm:"type:text;not null"`
	UpdatedAt time.Time
}

type Session struct {
	ID           string    `gorm:"primaryKey;size:64"` // sha256 hex of the cookie token
	UserID       uint64    `gorm:"not null;index"`
	Device       string    `gorm:"size:255;not null;default:''"`
	IP           string    `gorm:"size:64;not null;default:''"`
	Remember     bool      `gorm:"not null;default:false"`
	ExpiresAt    time.Time `gorm:"index"`
	CreatedAt    time.Time
	LastActiveAt time.Time
}

type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:64"`
	Value string `gorm:"type:text;not null"`
}

// MCPToken is the personal access token an AI client uses for /mcp.
// Only the SHA-256 hash is stored; each user has at most one token.
type MCPToken struct {
	ID         string `gorm:"primaryKey;size:64"` // sha256 hex of the token
	UserID     uint64 `gorm:"not null;uniqueIndex"`
	Hint       string `gorm:"size:32;not null;default:''"` // first characters, for display
	Timezone   string `gorm:"size:64;not null;default:''"` // browser timezone when generated
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// All returns every model for auto migration.
func All() []any {
	return []any{&User{}, &Card{}, &Item{}, &UserSetting{}, &Session{}, &SystemSetting{}, &MCPToken{}}
}
