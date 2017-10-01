package sdk

import (
	"google.golang.org/appengine/mail"
	"html/template"
	"bytes"
)

func (c *Context) SendEmail(message *mail.Message, t *template.Template, data interface{}) error {
	buf := new(bytes.Buffer)
	defer buf.Reset()
	err := t.ExecuteTemplate(buf, "email", data)
	if err != nil {
		return err
	}
	message.HTMLBody = buf.String()
	return mail.Send(c.Context, message)
}
