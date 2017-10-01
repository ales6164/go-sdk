package sdk

import (
	"github.com/asaskevich/govalidator"
	"time"
	"net/http"
	"errors"
	"github.com/google/uuid"
)

var UserEntity *Entity
var ProfileEntity *Entity
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
	UserEntity = NewEntity("user",
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
				Name:       "namespace",
				IsRequired: true,
				NoEdits:    true,
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
	ProfileEntity = NewEntity("profile",
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

	ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	d, err := ProfileEntity.Get(ctx, key)
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

	ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		return nil, err
	}

	d, err := ProfileEntity.Get(ctx, key)
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

	h, err := ProfileEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	key, err = ProfileEntity.Edit(ctx, key, h)
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
	d, err := UserEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	profileData, err := ProfileEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	ctx, key, err := UserEntity.NewKey(ctx, d.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	ctx, profileKey, err := ProfileEntity.NewKey(ctx, d.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	key, err = UserEntity.Add(ctx, key, d)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, nil, err
		}
		return id_token, nil, err
	}

	profileKey, err = ProfileEntity.Add(ctx, profileKey, profileData)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, nil, err
		}
		return id_token, nil, err
	}

	var data = d.Output()
	for name, value := range profileData.Output() {
		data[name] = value
	}

	id_token, err = NewToken(data["namespace"].(string), data["email"].(string))
	if err != nil {
		return id_token, nil, err
	}

	return id_token, data, err
}

func Login(ctx Context) (Token, map[string]interface{}, error) {
	var id_token Token

	if ctx.IsAuthenticated {
		return id_token, nil, ErrAlreadyAuthenticated
	}

	do, err := UserEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	ctx, key, err := UserEntity.NewKey(ctx, do.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	d, err := UserEntity.Get(ctx, key)
	if err != nil {
		return id_token, nil, err
	}

	err = decrypt([]byte(d.Get("password").([]uint8)), []byte(ctx.r.FormValue("password")))
	if err != nil {
		return id_token, nil, ErrNotAuthorized
	}

	ctx, profileKey, err := ProfileEntity.NewKey(ctx, do.Get("email"), false)
	if err != nil {
		return id_token, nil, err
	}

	profileD, err := ProfileEntity.Get(ctx, profileKey)
	if err != nil {
		return id_token, nil, err
	}

	var data = d.Output()
	for name, value := range profileD.Output() {
		data[name] = value
	}

	id_token, err = NewToken(d.Get("namespace").(string), d.Get("email").(string))
	if err != nil {
		return id_token, nil, err
	}
	return id_token, data, err
}

