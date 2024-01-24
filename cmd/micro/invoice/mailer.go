package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

//go:embed email-templates
var emailTemplateFS embed.FS

func (app *application) SendEmail(from, to, subject, tmpl string, attachments []string, data interface{}) error {
	templateToRender := fmt.Sprintf("email-templates/%s.html.tmpl", tmpl)
	t, err := template.New("email-html").ParseFS(emailTemplateFS, templateToRender)
	if err != nil {
		app.logger.Error(err.Error())
		return err
	}
	var tpl bytes.Buffer
	if err = t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.logger.Error(err.Error())
		return err
	}

	formattedMessage := tpl.String()

	templateToRender = fmt.Sprintf("email-templates/%s.plain.tmpl", tmpl)
	t, err = template.New("email-plain").ParseFS(emailTemplateFS, templateToRender)
	if err != nil {
		app.logger.Error(err.Error())
		return err
	}

	if err = t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.logger.Error(err.Error())
		return err
	}
	plainMessage := tpl.String()

	// app.logger.Info(fmt.Sprintf("formatted:\t %s\n", formattedMessage))
	// app.logger.Info(fmt.Sprintf("plain:\t %s\n", plainMessage))

	// send the mail
	server := mail.NewSMTPClient()
	server.Host = app.config.smtp.host
	server.Port = app.config.smtp.port
	server.Username = app.config.smtp.username
	server.Password = app.config.smtp.password
	server.Encryption = mail.EncryptionTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		return err
	}

	email := mail.NewMSG()
	email.SetFrom(from).AddTo(to).SetSubject(subject)

	email.SetBody(mail.TextHTML, formattedMessage)
	email.AddAlternative(mail.TextPlain, plainMessage)

	if len(attachments) > 0 {
		for _, x := range attachments {
			email.AddAttachment(x)
		}
	}

	err = email.Send(smtpClient)
	if err != nil {
		app.logger.Error(err.Error())
		return err
	}
	app.logger.Info("sent mail")

	return nil
}
