package outbound

import (
	"fmt"
	"net/smtp"
	"strings"

	"fsos-server/internal/domain/ports"
)

// SMTPEmailSender is a minimal ports.EmailSender over net/smtp with plain
// auth. Built by the factory only when SMTP_HOST is configured — everything
// email-dependent (users/recoverPassword) stays disabled otherwise.
type SMTPEmailSender struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPEmailSender(host, port, user, pass, from string) *SMTPEmailSender {
	return &SMTPEmailSender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTPEmailSender) SendEmail(msg *ports.EmailMessage) error {
	if s.host == "" {
		return fmt.Errorf("smtp not configured")
	}
	from := msg.From
	if from == "" {
		from = s.from
	}
	body := strings.Join([]string{
		"From: " + from,
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"",
		msg.Body,
	}, "\r\n")

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}
	return smtp.SendMail(s.host+":"+s.port, auth, from, []string{msg.To}, []byte(body))
}
