// Package deliver implements real out-of-app notification senders.
// In-app rows are always written first; external delivery is best-effort and
// never blocks or fails the reminder run. What is NOT claimed: WA delivery
// requires a gateway behind the webhook URL — we only prove handoff.
package deliver

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"
)

// Sender delivers one notification externally.
type Sender interface {
	Send(userID int64, contact Contact, title, body string) error
}

// Contact carries recipient addressing resolved from the users table.
type Contact struct {
	Name, Email, Phone string
}

// nop is the default: in-app only.
type nop struct{}

func (nop) Send(int64, Contact, string, string) error { return nil }

// Multi fans out to several senders, collecting the first error.
type Multi []Sender

func (m Multi) Send(uid int64, c Contact, title, body string) error {
	var first error
	for _, s := range m {
		if err := s.Send(uid, c, title, body); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Webhook posts signed JSON to a gateway URL (WA gateway, n8n, dsb).
// Signature: hex(HMAC-SHA256(secret, userID|title|body|ts)).
type Webhook struct {
	URL, Secret string
	Client      *http.Client
}

func (w Webhook) Send(uid int64, c Contact, title, body string) error {
	if w.URL == "" {
		return nil
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	mac := hmac.New(sha256.New, []byte(w.Secret))
	fmt.Fprintf(mac, "%d|%s|%s|%s", uid, title, body, ts)
	payload, _ := json.Marshal(map[string]any{
		"user_id": uid, "name": c.Name, "email": c.Email, "phone": c.Phone,
		"title": title, "body": body, "ts": ts,
	})
	req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	cl := w.Client
	if cl == nil {
		cl = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}

// SMTP sends plain-text email via env-configured relay.
type SMTP struct {
	Host, Port, User, Pass, From string
}

func (s SMTP) Send(_ int64, c Contact, title, body string) error {
	if s.Host == "" || c.Email == "" {
		return nil
	}
	addr := s.Host + ":" + s.Port
	msg := []byte("From: " + s.From + "\r\nTo: " + c.Email + "\r\nSubject: " + title +
		"\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body)
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}
	return smtp.SendMail(addr, auth, s.From, []string{c.Email}, msg)
}

// FromEnv builds the configured sender chain from environment.
// Empty env → in-app only (nop), which is always honest.
func FromEnv() Sender {
	var out Multi
	if url := os.Getenv("NOTARYOS_NOTIFY_WEBHOOK"); url != "" {
		out = append(out, Webhook{URL: url, Secret: os.Getenv("NOTARYOS_NOTIFY_SECRET")})
	}
	if host := os.Getenv("NOTARYOS_SMTP_HOST"); host != "" {
		port := os.Getenv("NOTARYOS_SMTP_PORT")
		if port == "" {
			port = "587"
		}
		out = append(out, SMTP{Host: host, Port: port, User: os.Getenv("NOTARYOS_SMTP_USER"),
			Pass: os.Getenv("NOTARYOS_SMTP_PASS"), From: os.Getenv("NOTARYOS_SMTP_FROM")})
	}
	if len(out) == 0 {
		return nop{}
	}
	return out
}
