package unit

import (
	"context"
	"strings"
	"testing"

	"goadmin/internal/config"
	"goadmin/internal/mail"
)

// New memilih LogMailer saat host kosong, SMTPMailer saat host diisi.
func TestMail_NewSelectsImpl(t *testing.T) {
	if _, ok := mail.New(config.MailConfig{}).(mail.LogMailer); !ok {
		t.Fatal("host kosong harus → LogMailer")
	}
	if _, ok := mail.New(config.MailConfig{Host: "smtp.example.com", Port: 587}).(*mail.SMTPMailer); !ok {
		t.Fatal("host terisi harus → *SMTPMailer")
	}
}

// LogMailer.Send tak error (dev fallback).
func TestMail_LogMailerSend(t *testing.T) {
	err := mail.LogMailer{}.Send(context.Background(), mail.Message{To: "a@b.com", Subject: "Hi", Body: "halo"})
	if err != nil {
		t.Fatalf("LogMailer.Send error: %v", err)
	}
}

// Build menyusun header RFC822 + body yang benar.
func TestMail_BuildMessage(t *testing.T) {
	raw := string(mail.Build("no-reply@goadmin.local", mail.Message{
		To: "user@example.com", Subject: "Reset Password", Body: "Kode: 123456",
	}))
	for _, want := range []string{
		"From: no-reply@goadmin.local",
		"To: user@example.com",
		"Subject: Reset Password",
		"Content-Type: text/plain",
		"Kode: 123456",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("pesan tak memuat %q\n---\n%s", want, raw)
		}
	}
	// Header & body dipisah baris kosong.
	if !strings.Contains(raw, "\r\n\r\n") {
		t.Fatal("pemisah header/body (CRLF CRLF) tak ada")
	}
}
