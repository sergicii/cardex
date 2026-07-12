package mailer

import (
	"fmt"
	"net/smtp"
)

type Mailer interface {
	SendVerificationCode(to, code string) error
}

type smtpMailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPMailer(host, port, username, password, from string) Mailer {
	return &smtpMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (m *smtpMailer) SendVerificationCode(to, code string) error {
	subject := "Código de verificación - Cardex"

	body := fmt.Sprintf(`Tu código de verificación es: %s

Este código expira en 5 minutos.

Si no solicitaste este código, puedes ignorar este mensaje.

Saludos,
Equipo Cardex`, code)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body)

	auth := smtp.PlainAuth("", m.username, m.password, m.host)
	addr := fmt.Sprintf("%s:%s", m.host, m.port)

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
	