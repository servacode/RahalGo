package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims محتوى توكن الوصول: هوية المستخدم وأدواره وعائلة جلسته.
// SID يجعل إبطال الجلسة فورياً: توكن الوصول عديم الحالة وصالح 15 دقيقة، فبدونه
// يبقى الخروج نافذاً على الورق فقط حتى انتهاء صلاحيته.
type Claims struct {
	Roles []string `json:"roles"`
	SID   string   `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenIssuer(secret string, accessTTL time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), accessTTL: accessTTL}
}

// AccessTTL عمر توكن الوصول — تحتاجه قائمة الإبطال لتضبط مدة بقائها.
func (t *TokenIssuer) AccessTTL() time.Duration { return t.accessTTL }

func (t *TokenIssuer) IssueAccess(userID string, roles []string, sessionID string) (string, time.Time, error) {
	exp := time.Now().Add(t.accessTTL)
	claims := Claims{
		Roles: roles,
		SID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "rahalgo",
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, exp, err
}

func (t *TokenIssuer) VerifyAccess(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("auth: unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("auth: invalid token")
	}
	return claims, nil
}

// NewOpaqueToken يولّد توكن تحديث عشوائياً (يُخزَّن مجزأً فقط).
func NewOpaqueToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
