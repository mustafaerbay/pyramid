package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// Password resets: a random token goes out by email; only its SHA-256 is
// stored. Links last an hour and work once.

const resetTTL = time.Hour

type passwordReset struct {
	TokenHash string    `gorm:"primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (passwordReset) TableName() string { return "password_resets" }

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// GetAuthOptions implements GET /api/v1/auth/options.
func (h *Handler) GetAuthOptions(c *gin.Context) {
	out := api.AuthOptions{EmailReset: h.mail != nil}
	if h.oidcOn() {
		on, label := true, h.cfg.OIDCLabel
		if label == "" {
			label = "Sign in with single sign-on"
		}
		out.Sso, out.SsoLabel = &on, &label
	}
	c.JSON(http.StatusOK, out)
}

// ForgotPassword implements POST /api/v1/auth/forgot. It answers the same
// whether or not the email has an account.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var in api.ForgotPassword
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	key := "forgot|" + normEmail(in.Email) + "|" + c.ClientIP()
	if h.logins.locked(key) {
		c.JSON(http.StatusTooManyRequests, api.Error{Message: "too many requests, try again in 15 minutes"})
		return
	}
	h.logins.fail(key) // every request counts toward the limit
	u, err := h.findUserByEmail(in.Email)
	if err != nil || !u.IsActive || h.mail == nil {
		c.Status(http.StatusNoContent)
		return
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "could not make a reset link"})
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if err := h.db.Create(&passwordReset{TokenHash: hashToken(token), UserID: u.ID, ExpiresAt: time.Now().Add(resetTTL)}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "could not make a reset link", Details: strptr(err.Error())})
		return
	}
	link := strings.TrimRight(h.cfg.PublicURL, "/") + "/reset?token=" + token
	body := "Someone asked to reset your Pyramid password.\n\nSet a new one here (the link works once, for an hour):\n" + link +
		"\n\nIf it wasn't you, you can ignore this email."
	go func(to string) {
		if err := h.mail(to, "Reset your Pyramid password", body); err != nil {
			log.Printf("reset email to %s failed: %v", to, err)
		}
	}(u.Email)
	c.Status(http.StatusNoContent)
}

// ResetPassword implements POST /api/v1/auth/reset.
func (h *Handler) ResetPassword(c *gin.Context) {
	var in api.ResetPassword
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	bad := func() {
		c.JSON(http.StatusBadRequest, api.Error{Message: "this link has expired or was already used; ask for a new one"})
	}
	var pr passwordReset
	if err := h.db.First(&pr, "token_hash = ?", hashToken(in.Token)).Error; err != nil || pr.UsedAt != nil || time.Now().After(pr.ExpiresAt) {
		bad()
		return
	}
	hash, err := hashPassword(in.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	now := time.Now().UTC()
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", pr.UserID).Update("password_hash", hash).Error; err != nil {
			return err
		}
		// Use this link and drop any other open ones for the account.
		return tx.Model(&passwordReset{}).Where("user_id = ? AND used_at IS NULL", pr.UserID).Update("used_at", now).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "could not set the password", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
