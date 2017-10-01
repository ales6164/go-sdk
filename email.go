package sdk

import (
	"google.golang.org/appengine/mail"
	"html/template"
	"bytes"
)

func (c *Context) SendEmail(message *mail.Message, t *template.Template, data interface{}) error {
	buf := new(bytes.Buffer)
	defer buf.Reset()
	t.ExecuteTemplate(buf, "email", data)
	message.HTMLBody = buf.String()
	return mail.Send(c.Context, message)
}
