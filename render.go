package sdk

type Render struct {
	Path    string                                                              `json:"path"` // "https://domain.com/:category/:id", "domain.com/other/{name}", ...
	URLFunc func(c Context, r Render, h *EntityDataHolder) (interface{}, error) `json:"-"`
}

/*func (a *SDK) publish(r *http.Request) {
	ctx := NewContext(r)

	appHostname := appengine.DefaultVersionHostname(ctx.Context)

	var sendValues = map[string]interface{}{
		"email":     holder.GetInput("email").(string),
		"signature": "varanox-admin",
	}

	bs, _ := json.Marshal(sendValues)

	client := urlfetch.Client(ctx.Context)
	resp, err := client.Post("https://"+holder.GetInput("domain").(string)+"/api/auth/client", "application/json", bytes.NewReader(bs))
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}
}*/

func (a *SDK) delete() {

}
