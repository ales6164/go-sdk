package sdk

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/mail"
	"net/http"
)

var clientIdSecret = &Entity{
	Name: "_client",
	Fields: []*Field{
		{
			Name:       "service",
			NoEdits:    true,
			NoIndex:    true,
			IsRequired: true,
			Validator: func(value interface{}) bool {
				return govalidator.IsByteLength(value.(string), 6, 128)
			},
		},
		{
			Name:       "secret",
			NoEdits:    true,
			IsRequired: true,
			NoIndex:    true,
			Validator: func(value interface{}) bool {
				return govalidator.IsByteLength(value.(string), 32, 128)
			},
		},
	},
}

var emptyEntity = &Entity{}

func NewClientRequest(a *SDK) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithScopes(ScopeRead, ScopeAdd)

		formHolder, _ := emptyEntity.FromForm(ctx)

		email := formHolder.GetInput("email")
		signature := formHolder.GetInput("signature")

		if email == nil || signature == nil {
			ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
			return
		}

		if !govalidator.IsEmail(email.(string)) || !govalidator.IsByteLength(signature.(string), 6, 128) {
			ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
			return
		}

		if a.AppOptions.AdminEmail != formHolder.GetInput("email").(string) {
			ctx.PrintError(w, ErrNotAuthorized, http.StatusInternalServerError)
			return
		}

		var secret = RandStringBytesMaskImprSrc(32)

		holder, err := clientIdSecret.FromMap(ctx, map[string]interface{}{
			"service": formHolder.GetInput("signature").(string),
			"secret":  secret,
		})
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx, key := clientIdSecret.NewIncompleteKey(ctx)

		key, err = clientIdSecret.Add(ctx, key, holder)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		err = sendClientSecret(ctx, formHolder.GetInput("email").(string), holder.id, secret)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, "ok")
	}
}

func IssueClientToken(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeRead, ScopeEdit)

	formHolder, err := emptyEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusBadRequest)
		return
	}

	ctx, key, err := clientIdSecret.DecodeKey(ctx, formHolder.GetInput("clientID").(string))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	holder, err := clientIdSecret.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, err, http.StatusUnauthorized)
		return
	}

	if holder.Get("secret").(string) != formHolder.GetInput("clientSecret").(string) {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusUnauthorized)
		return
	}

	err = ctx.NewUserToken(holder.id, APIClientRole)
	if err != nil {
		ctx.PrintError(w, err, http.StatusUnauthorized)
		return
	}

	ctx.Print(w, "ok")
}

func sendClientSecret(ctx Context, toEmail string, id string, secret string) error {
	return mail.Send(ctx.Context, &mail.Message{
		Sender:  "noreply@" + appengine.AppID(ctx.Context) + ".appspotmail.com",
		To:      []string{toEmail},
		Subject: "New Client Authorization Request",
		Body:    fmt.Sprintf("Client ID: %s\nClient Secret: %s", id, secret),
	})
}
