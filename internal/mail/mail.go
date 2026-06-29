// Package mail menyediakan pengiriman email yang dapat ditukar (interface),
// dengan implementasi SMTP (net/smtp) dan fallback LogMailer untuk dev/test
// (tak mengirim, hanya mencatat). Dipakai mis. untuk reset OTP / notifikasi.
package mail

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"goadmin/internal/config"
)

// Message = satu email teks-biasa.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer = kontrak pengirim email (mudah di-mock di test).
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// New memilih implementasi: SMTP bila host diset, selain itu LogMailer (dev).
func New(cfg config.MailConfig) Mailer {
	if strings.TrimSpace(cfg.Host) == "" {
		return LogMailer{}
	}
	return &SMTPMailer{cfg: cfg}
}

// LogMailer hanya mencatat (dev/test) — tak mengirim email nyata.
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf("[mail:log] to=%s subject=%q (MAIL_HOST belum dikonfigurasi)\n%s", msg.To, msg.Subject, msg.Body)
	return nil
}

// SMTPMailer mengirim via server SMTP (PLAIN auth).
type SMTPMailer struct {
	cfg config.MailConfig
}

func (m *SMTPMailer) Send(_ context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	from := m.cfg.FromAddress
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromAddress)
	}
	if err := smtp.SendMail(addr, auth, m.cfg.FromAddress, []string{msg.To}, Build(from, msg)); err != nil {
		return fmt.Errorf("mail: gagal kirim ke %s: %w", msg.To, err)
	}
	return nil
}

// Build menyusun pesan RFC 822 (header + body). Dipisah agar bisa diuji.
func Build(from string, msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	return []byte(b.String())
}
