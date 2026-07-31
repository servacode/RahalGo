package identity

import "time"

type User struct {
	ID          string   `json:"id"`
	Phone       string   `json:"phone"`
	FullName    string   `json:"full_name"`
	Status      string   `json:"status"`
	HasPassword bool     `json:"has_password"`
	InviteCode  *string  `json:"invite_code"`
	Roles       []string `json:"roles"`
	// MustChangePassword كلمة المرور وضعها طرف ثالث (مندوب/إدارة) — تُجبر الواجهة
	// صاحب الحساب على تبديلها قبل أي شاشة أخرى.
	MustChangePassword bool       `json:"must_change_password"`
	AvatarURL          *string    `json:"avatar_thumb_url"`
	LastSeenAt         *time.Time `json:"last_seen_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

type TokenPair struct {
	AccessToken     string    `json:"access_token"`
	AccessExpiresAt time.Time `json:"access_expires_at"`
	RefreshToken    string    `json:"refresh_token"`
}

type AuthResult struct {
	User   User      `json:"user"`
	Tokens TokenPair `json:"tokens"`
}
