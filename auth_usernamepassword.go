package sdk

import (
	"github.com/asaskevich/govalidator"
	"time"
	"net/http"
	"errors"
	"github.com/google/uuid"
	"html/template"
	"google.golang.org/appengine/mail"
)

var userEntity *Entity
var profileEntity *Entity
var lostPasswordRequest *Entity

func init() {
	lostPasswordRequest = NewEntity("lostPasswordRequest",
		[]*Field{
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
			{
				Name:         "isUnused",
				DefaultValue: true,
			},
		})
	userEntity = NewEntity("user",
		[]*Field{
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
			{
				Name:    "namespace",
				IsRequired: true,
				NoEdits: true,
				ValueFunc: func() interface{} {
					return uuid.New().String()
				},
			},
			{
				Name:       "password",
				IsRequired: true,
				NoIndex:    true,
				Json:       NoJsonOutput,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 6, 128)
				},
				TransformFunc: FuncHashTransform,
			},
		})
	profileEntity = NewEntity("profile",
		[]*Field{
			{
				Name:    "created",
				NoEdits: true,
				ValueFunc: func() interface{} {
					return time.Now()
				},
			},
			{
				Name:
				"firstName",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 1, 64)
				},
			},
			{
				Name:
				"lastName",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 1, 64)
				},
			},
			{
				Name: "companyName",
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 0, 64)
				},
			},
			{
				Name: "companyId",
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 0, 8)
				},
			},
			{
				Name:       "address",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 1, 128)
				},
			},
			{
				Name:       "city",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 1, 128)
				},
			},
			{
				Name:       "zip",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 1, 12) && govalidator.IsNumeric(value.(string))
				},
			},
			{
				Name: "phone",
			},
		})
}

func GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeGet)
	if !ctx.IsAuthenticated {
		ctx.PrintError(w, ErrNotAuthenticated, http.StatusUnauthorized)
		return
	}

	ctx, key, err := profileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	d, err := profileEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, d.Output())
}

func GetUserProfile(r *http.Request) (map[string]interface{}, error) {
	ctx := NewContext(r).WithScopes(ScopeGet)
	if !ctx.IsAuthenticated {
		return nil, ErrNotAuthenticated
	}

	ctx, key, err := profileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		return nil, err
	}

	d, err := profileEntity.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return d.Output(), nil
}

func EditUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeEdit)
	if ctx.err != nil {
		ctx.PrintError(w, ctx.err, http.StatusInternalServerError)
		return
	}
	if !ctx.IsAuthenticated {
		ctx.PrintError(w, ErrNotAuthenticated, http.StatusUnauthorized)
		return
	}

	h, err := profileEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, key, err := profileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	key, err = profileEntity.Edit(ctx, key, h)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, h.Output())
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeGet)
	id_token, profile, err := Login(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusUnauthorized)
		return
	}

	ctx.Token = id_token

	ctx.Print(w, profile)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeAdd)

	id_token, profile, err := Register(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusUnauthorized)
		return
	}

	ctx.Token = id_token

	ctx.Print(w, profile)
}

var (
	ErrAlreadyAuthenticated = errors.New("already authenticated")
)

func Register(ctx Context) (Token, map[string]interface{}, error) {
	var id_token Token

	if ctx.IsAuthenticated {
		return id_token, nil, ErrAlreadyAuthenticated
	}

	// Add user
	data, err := userEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	profileData, err := profileEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	ctx, key, err := userEntity.NewKey(ctx, data.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	ctx, profileKey, err := profileEntity.NewKey(ctx, data.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	key, err = userEntity.Add(ctx, key, data)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, nil, err
		}
		return id_token, nil, err
	}

	d := data.Output()

	profileKey, err = profileEntity.Add(ctx, profileKey, profileData)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, nil, err
		}
		return id_token, nil, err
	}

	for name, value := range profileData.Output() {
		d[name] = value
	}

	id_token, err = NewToken(d["namespace"].(string), d["email"].(string))
	if err != nil {
		return id_token, nil, err
	}

	return id_token, d, err
}

func Login(ctx Context) (Token, map[string]interface{}, error) {
	var id_token Token

	if ctx.IsAuthenticated {
		return id_token, nil, ErrAlreadyAuthenticated
	}

	do, err := userEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	ctx, key, err := userEntity.NewKey(ctx, do.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	d, err := userEntity.Get(ctx, key)
	if err != nil {
		return id_token, nil, err
	}

	err = decrypt([]byte(d.Get("password").([]uint8)), []byte(ctx.r.FormValue("password")))
	if err != nil {
		return id_token, nil, ErrNotAuthorized
	}

	ctx, profileKey, err := profileEntity.NewKey(ctx, do.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	profileD, err := profileEntity.Get(ctx, profileKey)
	if err != nil {
		return id_token, nil, err
	}

	for name, value := range profileD.Output() {
		d.AppendValue(name, value)
	}

	id_token, err = NewToken(d.Get("namespace").(string), d.Get("email").(string))
	if err != nil {
		return id_token, nil, err
	}
	return id_token, d.Output(), err
}

var recoverAccountEmailTemplate *template.Template

func init() {
	recoverAccountEmailTemplate, _ = template.New("").ParseFiles("lost_password.html")
}

func CreateLostPasswordRequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeGet, ScopeAdd)

	email := r.FormValue("email")
	if !govalidator.IsEmail(email) {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	ctx, key, err := userEntity.NewKey(ctx, email, false)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// check if user exists
	_, err = userEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	ctx, key = lostPasswordRequest.NewIncompleteKey(ctx, false)
	do, err := lostPasswordRequest.FromMap(ctx, map[string]interface{}{
		"email": email,
	})
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	key, err = lostPasswordRequest.Add(ctx, key, do)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// send recovery email with encoded key
	err = ctx.sendEmail(&mail.Message{
		Sender:  "Tisk Daril <no-reply@tiskdaril.appspotmail.com>",
		To:      []string{email},
		Subject: "Zahtevek za spremembo gesla",
	}, recoverAccountEmailTemplate, key.Encode())
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	ctx.Print(w, "success")
}

func UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeGet, ScopeEdit)

	// get form values
	encoded := r.FormValue("key")
	newPassword := r.FormValue("password")

	// decode key
	ctx, key, err := lostPasswordRequest.DecodeKey(ctx, encoded)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// get lost password datastore entry
	data, err := lostPasswordRequest.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// check if the request is still valid
	if !data.Get("isUnused").(bool) && !data.Get("created").(time.Time).After(time.Now().Add(-24 * time.Hour)) {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// set request invalid
	data.AppendValue("isUnused", false)
	do, err := lostPasswordRequest.FromMap(ctx, data.Output())
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}
	key, err = lostPasswordRequest.Edit(ctx, key, do)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// get user
	ctx, key, err = userEntity.NewKey(ctx, data.Get("email"), false)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}
	data, err = userEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// set new password
	data.AppendValue("password", newPassword)

	// update user
	do, err = userEntity.FromMap(ctx, data.Output())
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}
	key, err = userEntity.Edit(ctx, key, do)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	NewToken(do.Get("namespace").(string), do.Get("email").(string))
}
