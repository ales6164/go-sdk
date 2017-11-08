package sdk
/*
type Widget interface {
	WidgetName() string
}

func init() {
	gob.Register(time.Now())
	gob.Register(InputWidget{})
	gob.Register(MediaWidget{})
	gob.Register(SelectWidget{})
	gob.Register(ConnectWidget{})
	gob.Register(SummerNoteWidget{})
}

type InputWidget struct {
	Widget                `json:"-"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`
	Title        string   `json:"title"`
	Value        string   `json:"value"`
	Disabled     bool     `json:"disabled"`
	Readonly     bool     `json:"readonly"`
	Required     bool     `json:"required"`
	Placeholder  string   `json:"placeholder"`
	Pattern      string   `json:"pattern"`
	Step         int      `json:"step"`
	MinLength    int      `json:"minlength"`
	MaxLength    int      `json:"maxlength"`
	Min          int      `json:"min"`
	Max          int      `json:"max"`
	Size         int      `json:"size"`
	Autocomplete bool     `json:"autocomplete"`
	Autofocus    bool     `json:"autofocus"`
	List         []string `json:"list"`
}

type TextAreaWidget struct {
	Widget              `json:"-"`
	Label        string `json:"label"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Value        string `json:"value"`
	Disabled     bool   `json:"disabled"`
	Readonly     bool   `json:"readonly"`
	Required     bool   `json:"required"`
	Placeholder  string `json:"placeholder"`
	Pattern      string `json:"pattern"`
	Step         int    `json:"step"`
	MinLength    int    `json:"minlength"`
	MaxLength    int    `json:"maxlength"`
	Min          int    `json:"min"`
	Max          int    `json:"max"`
	Size         int    `json:"size"`
	Autocomplete bool   `json:"autocomplete"`
	Autofocus    bool   `json:"autofocus"`
}

type SummerNoteWidget struct {
	Widget       `json:"-"`
	Label string `json:"label"`
}

type SelectWidget struct {
	Widget                `json:"-"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`
	Title        string   `json:"title"`
	Value        string   `json:"value"`
	Disabled     bool     `json:"disabled"`
	Readonly     bool     `json:"readonly"`
	Required     bool     `json:"required"`
	Placeholder  string   `json:"placeholder"`
	Pattern      string   `json:"pattern"`
	Step         int      `json:"step"`
	MinLength    int      `json:"minlength"`
	MaxLength    int      `json:"maxlength"`
	Min          int      `json:"min"`
	Max          int      `json:"max"`
	Size         int      `json:"size"`
	Autocomplete bool     `json:"autocomplete"`
	Autofocus    bool     `json:"autofocus"`
	List         []string `json:"list"`
}

type ConnectWidget struct {
	Widget                          `json:"-"`
	Label    string                 `json:"label"`
	Entity   string                 `json:"entity"`
	Meta     map[string]interface{} `json:"meta"`
	Type     string                 `json:"type"`     // select
	ReadOnly bool                   `json:"readOnly"` // if false, can also add new entities
}

// media widget example
type MediaWidget struct {
	Widget                      `json:"-"`
	Label         string        `json:"label"`
	Multiple      string        `json:"-"`
	Type          string        `json:"-"`
	TransformTool TransformTool `json:"-"`
}

type TransformTool struct {
	CropToSizes []string // evolve
	// ... and others
}

func (w SelectWidget) WidgetName() string {
	return "select"
}

func (w InputWidget) WidgetName() string {
	return "input"
}

func (w TextAreaWidget) WidgetName() string {
	return "textarea"
}

func (w ConnectWidget) WidgetName() string {
	return "connect"
}

func (w MediaWidget) WidgetName() string {
	return "media"
}

func (w SummerNoteWidget) WidgetName() string {
	return "summernote"
}*/
