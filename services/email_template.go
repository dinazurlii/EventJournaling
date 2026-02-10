package services

import (
	"bytes"
	"html/template"
)

func RenderEmailTemplate(name string, data any) (string, error) {
	tmpl, err := template.ParseFiles("templates/emails/" + name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
