package ports

type EmailMessage struct {
	From     string
	To       string
	Subject  string
	SMTPHost string
	Body     string
}

type EmailSender interface {
	SendEmail(msg *EmailMessage) error
}
