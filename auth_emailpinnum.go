package sdk

import (
	"github.com/asaskevich/govalidator"
	"time"
	"net/http"
	"google.golang.org/appengine/mail"
	"errors"
	"google.golang.org/appengine"
)

/*var iamuser = NewEntity("iamuser", []*Field{
	{
		Name:    "created",
		NoEdits: true,
		ValueFunc: func() interface{} {
			return time.Now()
		},
	},
	{
		Name:       "email",
		NoEdits:    true,
		IsRequired: true,
		Validator: func(value interface{}) bool {
			return govalidator.IsEmail(value.(string))
		},
	},
})*/

var iamuserVerification = NewEntity("iamuser_pin", []*Field{
	{
		Name:    "created",
		NoEdits: true,
		ValueFunc: func() interface{} {
			return time.Now()
		},
	},
	{
		Name:    "expires",
		NoEdits: true,
		ValueFunc: func() interface{} {
			return time.Now().Add(time.Minute * 15)
		},
	},
	{
		Name: "used",
	},
	{
		Name:       "email",
		NoEdits:    true,
		IsRequired: true,
		Validator: func(value interface{}) bool {
			return govalidator.IsEmail(value.(string))
		},
	},
	{
		Name:       "pin",
		IsRequired: true,
		NoIndex:    true,
		Json:       NoJsonOutput,
		Validator: func(value interface{}) bool {
			return govalidator.IsByteLength(value.(string), 6, 128)
		},
		TransformFunc: FuncHashTransform,
	},
})

func AuthWithIAmUser(a *SDK) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithScopes(ScopeGet, ScopeAdd)

		holder, err := iamuserVerification.FromForm(ctx)
		if err != nil {
			ctx.PrintError(w, err, http.StatusBadRequest)
			return
		}

		if a.AppOptions.AdminEmail != holder.GetInput("email") {
			ctx.PrintError(w, ErrNotAuthorized, http.StatusInternalServerError)
			return
		}

		ctx, key := iamuserVerification.NewIncompleteKey(ctx, false)

		var pin = RandNumberBytesMaskImprSrc(6)
		holder.AppendValue("pin", pin)
		holder.AppendValue("used", false)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		_, err = iamuserVerification.Add(ctx, key, holder)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		err = sendPinNum(ctx, holder.GetInput("email").(string), pin)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, "ok")
	}
}

func VerifyIAmUser(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeGet, ScopeEdit)

	input, err := iamuserVerification.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusBadRequest)
		return
	}

	holders, err := iamuserVerification.Query(ctx, "", "", 1,
		EntityQueryFilter{Name: "email", Value: input.GetInput("email"), Operator: "="},
		EntityQueryFilter{Name: "expires", Value: time.Now(), Operator: ">"},
		EntityQueryFilter{Name: "used", Value: false, Operator: "="},
	)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}
	if len(holders) == 0 {
		ctx.PrintError(w, errors.New("got null"), http.StatusUnauthorized)
		return
	}

	var verified bool
	for _, holder := range holders {
		err = decrypt([]byte(holder.Get("pin").([]uint8)), []byte(input.GetInput("pin").(string)))
		if err == nil {
			ctx, key, err := iamuserVerification.DecodeKey(ctx, holder.id)
			if err != nil {
				ctx.PrintError(w, err, http.StatusUnauthorized)
				return
			}

			holder.AppendValue("used", true)

			_, err = iamuserVerification.Edit(ctx, key, holder)
			if err != nil {
				ctx.PrintError(w, err, http.StatusUnauthorized)
				return
			}

			verified = true
			break
		}
	}

	if !verified {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusUnauthorized)
		return
	}

	id_token, err := NewToken("", input.GetInput("email").(string))
	if err != nil {
		ctx.PrintError(w, err, http.StatusUnauthorized)
		return
	}

	ctx.Token = id_token

	ctx.Print(w, "ok")
}

func sendPinNum(ctx Context, toEmail string, pin string) error {
	return mail.Send(ctx.Context, &mail.Message{
		Sender:  "noreply@" + appengine.AppID(ctx.Context) + ".appspotmail.com",
		To:      []string{toEmail},
		Subject: "New Authorization Request",
		Body:    pin,
	})
}
