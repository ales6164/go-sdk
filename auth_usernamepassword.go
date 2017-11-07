package sdk

import (
	"errors"
	"github.com/asaskevich/govalidator"
	"net/http"
)

var userEntity *Entity
var ProfileEntity *Entity
var lostPasswordRequest *Entity

func init() {
	lostPasswordRequest = &Entity{
		Name: "lostPasswordRequest",
		Fields: []*Field{
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
		},
	}
	userEntity = &Entity{
		Name: "user",
		Fields: []*Field{
			{
				Name:       "email",
				NoEdits:    true,
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsEmail(value.(string))
				},
			},
			{
				Name:         "role",
				DefaultValue: SubscriberRole,
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
		},
	}
	ProfileEntity = &Entity{
		Name: "profile",
		Fields: []*Field{
			{
				Name:       "firstName",
				IsRequired: true,
				Validator: func(value interface{}) bool {
					return govalidator.IsByteLength(value.(string), 1, 64)
				},
			},
			{
				Name:       "lastName",
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
			{
				Name: "email",
				Validator: func(value interface{}) bool {
					return govalidator.IsEmail(value.(string))
				},
			},
		},
	}
}

func GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeRead)
	if !ctx.IsAuthenticated {
		ctx.PrintError(w, ErrNotAuthenticated, http.StatusUnauthorized)
		return
	}

	ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	d, err := ProfileEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, d.Output(ctx))
}

func GetUserProfile(r *http.Request) (map[string]interface{}, error) {
	ctx := NewContext(r).WithScopes(ScopeRead)
	if !ctx.IsAuthenticated {
		return nil, ErrNotAuthenticated
	}

	ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User)
	if err != nil {
		return nil, err
	}

	d, err := ProfileEntity.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return d.Output(ctx), nil
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

	ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	key, err = ProfileEntity.Edit(ctx, key, h)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, h.Output(ctx))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeRead)

	if ctx.IsAuthenticated {
		ctx.PrintError(w, ErrAlreadyAuthenticated, http.StatusInternalServerError)
		return
	}

	do, err := userEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, key, err := userEntity.NewKey(ctx, do.GetInput("email"))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	d, err := userEntity.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	err = decrypt([]byte(d.Get("password").([]uint8)), []byte(do.GetInput("password").(string)))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, profileKey, err := ProfileEntity.NewKey(ctx, d.id)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	profileD, err := ProfileEntity.Get(ctx, profileKey)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	var data = d.Output(ctx)
	for name, value := range profileD.Output(ctx) {
		data[name] = value
	}

	err = ctx.NewUserToken(d.id, Role(d.Get("role").(string)))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, data)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r).WithScopes(ScopeAdd)

	if ctx.IsAuthenticated {
		ctx.PrintError(w, ErrAlreadyAuthenticated, http.StatusInternalServerError)
		return
	}

	// Add user
	d, err := userEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	profileData, err := ProfileEntity.FromForm(ctx)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, key, err := userEntity.NewKey(ctx, d.Get("email"))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	key, err = userEntity.Add(ctx, key, d)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx, profileKey, err := ProfileEntity.NewKey(ctx, d.id)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	profileKey, err = ProfileEntity.Add(ctx, profileKey, profileData)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	var data = d.Output(ctx)

	err = ctx.NewUserToken(d.id, Role(d.Get("role").(string)))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	for name, value := range profileData.Output(ctx) {
		data[name] = value
	}

	ctx.Print(w, data)
}

var (
	ErrAlreadyAuthenticated = errors.New("already authenticated")
)
