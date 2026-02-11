package services

import (
	"bytes"
	"crypto/tls"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strconv"
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

func SendEventApproved(toEmail string, title string, eventID int) error {
	log.Println("Sending approved email to:", toEmail)

	t, err := template.ParseFiles("templates/emails/approved.html")
	if err != nil {
		log.Println("Template error:", err)
		return err
	}

	var body bytes.Buffer

	data := map[string]any{
		"Title":    title,
		"EventURL": os.Getenv("FRONTEND_URL") + "/events/" + strconv.Itoa(eventID),
	}

	err = t.Execute(&body, data)
	if err != nil {
		log.Println("Execute template error:", err)
		return err
	}

	err = SendEmail(toEmail, "Your Event Was Approved 🎉", body.String())
	if err != nil {
		log.Println("Send mail error:", err)
		return err
	}

	log.Println("Email sent successfully!")
	return nil
}

func SendEventRejected(toEmail string, title string, reason string, eventID int) error {

	t, err := template.ParseFiles("templates/emails/rejected.html")
	if err != nil {
		return err
	}

	var body bytes.Buffer

	data := map[string]any{
		"Title":    title,
		"Reason":   reason,
		"EventURL": os.Getenv("FRONTEND_URL") + "/events/" + strconv.Itoa(eventID),
	}

	err = t.Execute(&body, data)
	if err != nil {
		return err
	}

	return SendEmail(toEmail, "Your Event Was Rejected ❌", body.String())
}
