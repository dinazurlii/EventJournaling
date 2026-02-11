package services

import (
	"crypto/tls"
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

	tlsconfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", host+":"+port, tlsconfig)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}

	if err = client.Auth(auth); err != nil {
		return err
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	if err = client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

func SendEventApproved(to string, title string) {
	subject := "Your event has been approved 🎉"

	body := `
		<h2>Event Approved</h2>
		<p>Your event <strong>` + title + `</strong> has been approved by admin.</p>
		<p>It is now publicly visible.</p>
	`

	_ = SendEmail(to, subject, body)
}

func SendEventRejected(to string, title string, reason string) {
	subject := "Your event has been rejected ❌"

	body := `
		<h2>Event Rejected</h2>
		<p>Your event <strong>` + title + `</strong> was rejected.</p>
		<p><strong>Reason:</strong> ` + reason + `</p>
		<p>Please revise and resubmit.</p>
	`

	_ = SendEmail(to, subject, body)
}
