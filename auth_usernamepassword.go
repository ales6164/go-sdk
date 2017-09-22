package sdk

import (
	"github.com/asaskevich/govalidator"
	"time"
	"net/http"
	"errors"
	"github.com/google/uuid"
)

var userEntity *PreparedEntity

func init() {
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
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 6, 128)
				},
				TransformFunc: FuncHashTransform,
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

func GetUserProfile(r *http.Request) (map[string]interface{}, error) {
	ctx := NewContext(r).WithScopes(ScopeGet)
	if !ctx.IsAuthenticated {
		return nil, ErrNotAuthenticated
	}

	ctx, key, err := userEntity.NewKey(ctx, ctx.User, false)
	if err != nil {
		return nil, err
	}

	ps, err := userEntity.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	d := userEntity.GetOutputData(ps)
	delete(d, "password")

	return d, nil
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

	data, err := userEntity.FromForm(ctx)
	if err != nil {
		return id_token, nil, err
	}

	ctx, key, err := userEntity.NewKey(ctx, data.Input["email"], false)
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
	delete(d, "password")

	id_token, err = NewToken(d["namespace"].(string), d["email"].(string))
	return id_token, d, err
}

func Login(ctx Context) (Token, map[string]interface{}, error) {
	var id_token Token

	if ctx.IsAuthenticated {
		return id_token, nil, ErrAlreadyAuthenticated
	}

	username := ctx.r.FormValue("email")
	password := ctx.r.FormValue("password")
	if len(username) == 0 {
		return id_token, nil, FieldRequired.Params("email")
	}

	ctx, key, err := userEntity.NewKey(ctx, username, false)
	if err != nil {
		return id_token, nil, err
	}

	ps, err := userEntity.Get(ctx, key)
	if err != nil {
		return id_token, nil, err
	}

	d := userEntity.GetOutputData(ps)
	err = decrypt([]byte(d["password"].([]uint8)), []byte(password))
	if err != nil {
		return id_token, nil, ErrNotAuthorized
	}
	delete(d, "password")

	id_token, err = NewToken(d["namespace"].(string), d["email"].(string))
	return id_token, d, err
}

func (a *SDK) UpdatePassword(ctx Context) (bool, error) {
	/*var newPassword interface{}

	*//*email, ok := GetUser(ctx.r)
	if !ok {
		return false, errors.New("invalid token")
	}*//*

	engineCtx, key, d, err := userEntity.FromForm(ctx, false)
	if err != nil {
		return false, err
	}

	var ok bool
	if newPassword, ok = d.Input["newPassword"]; !ok {
		return false, errors.New("field newPassword is empty")
	}

	err = Get(engineCtx, key, &d.Output)
	if err != nil {
		return false, err
	}

	for _, prop := range d.Output {
		if prop.Name == "password" {
			prop.Value = newPassword
		}
	}

	_, err = Put(engineCtx, key, d.Output)
	if err != nil {
		return false, err
	}*/

	return true, nil
}
