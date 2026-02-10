package services

import (
	"net/smtp"
	"os"
)

func SendEmail(to, subject, body string) error {
	from := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	fromName := os.Getenv("FROM_EMAIL")

	auth := smtp.PlainAuth("", from, password, host)

	msg := []byte(
		"From: " + fromName + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n" +
			body,
	)

	return smtp.SendMail(host+":"+port, auth, from, []string{to}, msg)
}
