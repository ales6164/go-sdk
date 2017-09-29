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

var userEntity *PreparedEntity
var profileEntity *PreparedEntity
var lostPasswordRequest *PreparedEntity

func init() {
	lostPasswordRequest = NewEntity("lostPasswordRequest",
		[]*Field{
			{
				Name: "created",
				WithValueFunc: func() interface{} {
					return time.Now()
				},
			},
			{
				Name:       "email",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsEmail(value.(string))
				},
			},
			{
				Name:         "isUnused",
				DefaultValue: true,
			},
		}).Prepare()
	userEntity = NewEntity("user",
		[]*Field{
			{
				Name: "created",
				WithValueFunc: func() interface{} {
					return time.Now()
				},
			},
			{
				Name:
				"email",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsEmail(value.(string))
				},
			},
			{
				Name: "namespace",
				WithValueFunc: func() interface{} {
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
		}).Prepare()
	profileEntity = NewEntity("profile",
		[]*Field{
			{
				Name: "created",
				WithValueFunc: func() interface{} {
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
		}).Prepare()
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

	d, _, err := profileEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, d)
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

	d, _, err := profileEntity.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return d, nil
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

	do, err := profileEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, key, err := profileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	key, err = profileEntity.Edit(ctx, key, do.Output)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	d := profileEntity.GetOutputData(do.Output)
	ctx.Print(w, d)
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

	ctx, key, err := userEntity.NewKey(ctx, data.DataMap["email"], false)
	if err != nil {
		return id_token, nil, err
	}

	ctx, profileKey, err := profileEntity.NewKey(ctx, data.DataMap["email"], false)
	if err != nil {
		return id_token, nil, err
	}

	key, err = userEntity.Add(ctx, key, data.Output)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, nil, err
		}
		return id_token, nil, err
	}

	d := userEntity.GetOutputData(data.Output)

	profileKey, err = profileEntity.Add(ctx, profileKey, profileData.Output)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, nil, err
		}
		return id_token, nil, err
	}

	for name, value := range profileEntity.GetOutputData(profileData.Output) {
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

	ctx, key, err := userEntity.NewKey(ctx, do.DataMap["email"], false)
	if err != nil {
		return id_token, nil, err
	}

	_, ps, err := userEntity.Get(ctx, key)
	if err != nil {
		return id_token, nil, err
	}

	d := map[string]interface{}{}
	for _, val := range ps {
		d[val.Name] = val.Value
	}
	err = decrypt([]byte(d["password"].([]uint8)), []byte(ctx.r.FormValue("password")))
	if err != nil {
		return id_token, nil, ErrNotAuthorized
	}
	delete(d, "password")

	ctx, profileKey, err := profileEntity.NewKey(ctx, do.DataMap["email"], false)
	if err != nil {
		return id_token, nil, err
	}

	profileD, ps, err := profileEntity.Get(ctx, profileKey)
	if err != nil {
		return id_token, nil, err
	}

	for name, value := range profileD {
		d[name] = value
	}

	id_token, err = NewToken(d["namespace"].(string), d["email"].(string))
	if err != nil {
		return id_token, nil, err
	}
	return id_token, d, err
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
	_, _, err = userEntity.Get(ctx, key)
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

	key, err = lostPasswordRequest.Add(ctx, key, do.Output)
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
	data, ps, err := lostPasswordRequest.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// check if the request is still valid
	if !data["isUnused"].(bool) && !data["created"].(time.Time).After(time.Now().Add(-24 * time.Hour)) {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// set request invalid
	data["isUnused"] = false
	do, err := lostPasswordRequest.FromMap(ctx, data)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}
	key, err = lostPasswordRequest.Edit(ctx, key, do.Output)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// get user
	ctx, key, err = userEntity.NewKey(ctx, data["email"], false)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}
	data, ps, err = userEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	// set new password
	data["password"] = newPassword

	// update user
	do, err = userEntity.FromMap(ctx, data)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}
	key, err = userEntity.Edit(ctx, key, do.Output)
	if err != nil {
		ctx.PrintError(w, ErrNotAuthorized, http.StatusBadRequest)
		return
	}

	NewToken(do.DataMap["namespace"])
}
