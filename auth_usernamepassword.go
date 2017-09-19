package sdk

import (
	"github.com/asaskevich/govalidator"
	"time"
	"net/http"
	"errors"
)

var userEntity *PreparedEntity

type fun func(func()) interface{}

func init() {
	userEnt := NewEntity("user", []*Field{
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
			Name:       "password",
			IsRequired: true,
			NoIndex:    true,
			Json:       NoJsonOutput,
			Validator: func(value interface{}) bool {
				return govalidator.IsByteLength(value.(string), 6, 128)
			},
			TransformFunc: FuncHashTransform,
		},
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
	})
	userEnt.Key = &Key{
		Kind:          "User",
		FromField:     "email",
		FromToken:     true,
		NamespaceType: NoNamespace,
	}
	userEnt.Rules = Rules{
		GuestAdd:  true,
		GuestRead: true,
		UserEdit:  true,
	}
	userEnt.OnRead = func(c *Conn) error {
		var data = c.Entity.GetData()
		return decrypt([]byte(data["password"].([]uint8)), []byte(c.InputData["password"][0].(string)))
	}
	userEntity = PrepareEntity(userEnt)
}

func (a *SDK) CheckToken(w http.ResponseWriter, r *http.Request) {
	_, ok := GetUser(r)
	if ok {
		printData(w, "valid")
		return
	}
	printError(w, errors.New(""), http.StatusUnauthorized)
}

func (a *SDK) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// todo: wtf? fix
	r.FormValue("ada")

	ctx := NewContext(r)
	id_token, data, err := a.Login(ctx)
	if err != nil {
		printError(w, err, http.StatusUnauthorized)
		return
	}

	printData(w, map[string]interface{}{
		"id_token": id_token,
		"profile":  data,
	})
}

func (a *SDK) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r)

	// todo: wtf? fix
	r.FormValue("ada")

	id_token, data, err := a.Register(ctx)
	if err != nil {
		printError(w, err, http.StatusUnauthorized)
		return
	}

	printData(w, map[string]interface{}{
		"id_token": id_token,
		"profile":  data,
	})
}

func (a *SDK) Register(ctx Context) (string, map[string]interface{}, error) {
	var id_token string

	engineCtx, key, data, err := userEntity.FromForm(ctx, true)
	if err != nil {
		return id_token, data.Input, err
	}

	key, err = Post(engineCtx, key, data.Output)
	if err != nil {
		if err == EntityAlreadyExists {
			// todo
			return id_token, data.Input, err
		}
		return id_token, data.Input, err
	}

	err = Get(engineCtx, key, &data.Output)
	if err != nil {
		return id_token, data.Input, err
	}

	d := userEntity.GetOutputData(data.Output)

	token, err := a.NewToken(d["email"].(string))
	return token, d, err
}

func (a *SDK) Login(ctx Context) (string, map[string]interface{}, error) {
	var id_token string

	engineCtx, key, d, err := userEntity.FromForm(ctx, true)
	if err != nil {
		return id_token, d.Input, err
	}

	err = Get(engineCtx, key, &d.Output)
	if err != nil {
		return id_token, d.Input, err
	}

	data := userEntity.GetOutputData(d.Output)

	token, err := a.NewToken(data["email"].(string))
	return token, data, err
}

func (a *SDK) UpdatePassword(ctx Context) (bool, error) {
	var newPassword interface{}

	/*email, ok := GetUser(ctx.r)
	if !ok {
		return false, errors.New("invalid token")
	}*/

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
	}

	return true, nil
}
