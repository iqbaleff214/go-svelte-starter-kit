package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"

	"github.com/404nfid/go-svelte-starter-kit/pkg/config"
)

// Message is the data needed to send one email.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Sender is the interface both SMTP and SendGrid adapters implement.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// NewSender returns an SMTP or SendGrid sender based on config.
func NewSender(cfg config.EmailConfig) Sender {
	if cfg.Provider == "sendgrid" {
		return &sendGridSender{apiKey: cfg.SendGridKey, from: cfg.SMTPFrom}
	}
	return &smtpSender{cfg: cfg}
}

// ---- SMTP ----

type smtpSender struct{ cfg config.EmailConfig }

func (s *smtpSender) Send(_ context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)

	body := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.cfg.SMTPFrom, msg.To, msg.Subject, msg.HTML,
	)

	return smtp.SendMail(addr, auth, s.cfg.SMTPFrom, []string{msg.To}, []byte(body))
}

// ---- SendGrid ----

type sendGridSender struct{ apiKey, from string }

func (s *sendGridSender) Send(ctx context.Context, msg Message) error {
	payload, _ := json.Marshal(map[string]any{
		"personalizations": []map[string]any{{"to": []map[string]string{{"email": msg.To}}}},
		"from":             map[string]string{"email": s.from},
		"subject":          msg.Subject,
		"content": []map[string]string{
			{"type": "text/html", "value": msg.HTML},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.sendgrid.com/v3/mail/send", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
	}
	return nil
}
