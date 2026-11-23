package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Two-factor sign-in with TOTP (RFC 6238: 6 digits, 30 s, SHA-1 — what
// authenticator apps use), plus ten one-time recovery codes.

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func decodeSecret(s string) ([]byte, error) {
	return b32.DecodeString(strings.ToUpper(strings.ReplaceAll(s, " ", "")))
}

// totpAt is the 6-digit code for one 30-second step.
func totpAt(key []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	m := hmac.New(sha1.New, key)
	m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	n := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", n%1000000)
}

type totpState struct {
	Secret   *string
	Enabled  bool
	Recovery string
	LastStep int64
}

func (h *Handler) totpOf(userID uuid.UUID) totpState {
	var s totpState
	h.db.Table("users").Select("totp_secret, totp_enabled, COALESCE(totp_recovery, '[]'), totp_last_step").Where("id = ?", userID).Row().
		Scan(&s.Secret, &s.Enabled, &s.Recovery, &s.LastStep)
	return s
}

// checkTOTP accepts a code from the previous, current or next step that's
// newer than the last one used, and returns that step.
func checkTOTP(secret string, code string, lastStep int64) (int64, bool) {
	key, err := decodeSecret(secret)
	code = strings.ReplaceAll(code, " ", "")
	if err != nil || len(code) != 6 {
		return 0, false
	}
	now := time.Now().Unix() / 30
	for _, st := range []int64{now - 1, now, now + 1} {
		if st > lastStep && hmac.Equal([]byte(totpAt(key, st)), []byte(code)) {
			return st, true
		}
	}
	return 0, false
}

// secondFactor checks a TOTP or recovery code for a user with 2FA on.
func (h *Handler) secondFactor(u models.User, code string) bool {
	s := h.totpOf(u.ID)
	if s.Secret == nil {
		return false
	}
	if st, ok := checkTOTP(*s.Secret, code, s.LastStep); ok {
		h.db.Table("users").Where("id = ?", u.ID).Update("totp_last_step", st)
		return true
	}
	var hashes []string
	_ = json.Unmarshal([]byte(s.Recovery), &hashes)
	norm := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	want := hashToken(norm)
	for i, hsh := range hashes {
		if hmac.Equal([]byte(hsh), []byte(want)) {
			hashes = append(hashes[:i], hashes[i+1:]...)
			b, _ := json.Marshal(hashes)
			h.db.Table("users").Where("id = ?", u.ID).Update("totp_recovery", string(b))
			return true
		}
	}
	return false
}

// GetTwoFactor implements GET /api/v1/auth/2fa.
func (h *Handler) GetTwoFactor(c *gin.Context) {
	me, _ := currentUser(c)
	s := h.totpOf(me.ID)
	var hashes []string
	_ = json.Unmarshal([]byte(s.Recovery), &hashes)
	c.JSON(http.StatusOK, api.TwoFactorStatus{Enabled: s.Enabled, RecoveryCodesLeft: len(hashes)})
}

// SetupTwoFactor implements POST /api/v1/auth/2fa/setup.
func (h *Handler) SetupTwoFactor(c *gin.Context) {
	me, ok := currentUser(c)
	if !ok || viaToken(c) {
		c.JSON(http.StatusForbidden, api.Error{Message: "sign in to change two-factor sign-in"})
		return
	}
	if h.totpOf(me.ID).Enabled {
		c.JSON(http.StatusBadRequest, api.Error{Message: "two-factor sign-in is already on; turn it off first"})
		return
	}
	buf := make([]byte, 20)
	_, _ = rand.Read(buf)
	secret := b32.EncodeToString(buf)
	h.db.Table("users").Where("id = ?", me.ID).Update("totp_secret", secret)
	label := url.PathEscape("Pyramid:" + me.Email)
	c.JSON(http.StatusOK, api.TwoFactorSetup{Secret: secret, OtpauthUrl: fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=Pyramid&digits=6&period=30", label, secret)})
}

// EnableTwoFactor implements POST /api/v1/auth/2fa/enable.
func (h *Handler) EnableTwoFactor(c *gin.Context) {
	me, ok := currentUser(c)
	if !ok || viaToken(c) {
		c.JSON(http.StatusForbidden, api.Error{Message: "sign in to change two-factor sign-in"})
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	_ = c.ShouldBindJSON(&in)
	s := h.totpOf(me.ID)
	if s.Secret == nil || s.Enabled {
		c.JSON(http.StatusBadRequest, api.Error{Message: "set it up first"})
		return
	}
	st, ok := checkTOTP(*s.Secret, in.Code, 0)
	if !ok {
		c.JSON(http.StatusBadRequest, api.Error{Message: "that code doesn't match; check the time on your phone and try the next code"})
		return
	}
	codes := make([]string, 10)
	hashes := make([]string, 10)
	for i := range codes {
		b := make([]byte, 5)
		_, _ = rand.Read(b)
		x := hex.EncodeToString(b)
		codes[i] = x[:5] + "-" + x[5:]
		hashes[i] = hashToken(strings.ReplaceAll(codes[i], " ", ""))
	}
	hb, _ := json.Marshal(hashes)
	h.db.Table("users").Where("id = ?", me.ID).Updates(map[string]any{"totp_enabled": true, "totp_recovery": string(hb), "totp_last_step": st})
	h.audit(c, &me, me.Email, "turn on two-factor sign-in", "user", strptr(me.ID.String()), nil)
	c.JSON(http.StatusOK, api.TwoFactorRecovery{RecoveryCodes: codes})
}

func (h *Handler) clearTOTP(id uuid.UUID) {
	h.db.Table("users").Where("id = ?", id).Updates(map[string]any{"totp_enabled": false, "totp_secret": nil, "totp_recovery": "[]", "totp_last_step": 0})
}

// DisableTwoFactor implements POST /api/v1/auth/2fa/disable.
func (h *Handler) DisableTwoFactor(c *gin.Context) {
	me, ok := currentUser(c)
	if !ok || viaToken(c) {
		c.JSON(http.StatusForbidden, api.Error{Message: "sign in to change two-factor sign-in"})
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&in)
	if me.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*me.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "wrong password"})
		return
	}
	h.clearTOTP(me.ID)
	c.Status(http.StatusNoContent)
}

// ResetUserTwoFactor implements POST /api/v1/users/{id}/reset-2fa.
func (h *Handler) ResetUserTwoFactor(c *gin.Context) {
	var u models.User
	if err := h.db.First(&u, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "account not found"})
		return
	}
	h.clearTOTP(u.ID)
	c.Status(http.StatusNoContent)
}
