package sdk

//type role map[Role]map[Scope]bool

//var userRoles = map[string]role{}

type Role string

const (
	GuestRole      Role = "guest" // Public access
	AdminRole      Role = "admin"
	APIClientRole  Role = "api_client"
	SubscriberRole Role = "subscriber" // Default user role
)

func (c Context) HasScope(e *Entity, scope Scope) bool {
	if c.scopes != nil {
		if s, ok := c.scopes[scope]; ok {
			return s
		}
	}

	if role, ok := e.Rules[c.Role]; ok {
		if s, ok := role[scope]; ok {
			return s
		}
	}

	return false
}

func (c Context) WithScopes(scopes ...Scope) Context {
	c.scopes = map[Scope]bool{}
	for _, scope := range scopes {
		c.scopes[scope] = true
	}
	return c
}

func (e *Entity) SetRule(role Role, scopes ...Scope) {
	if e.Rules == nil {
		e.Rules = map[Role]map[Scope]bool{}
	}
	if e.Rules[role] == nil {
		e.Rules[role] = map[Scope]bool{}
	}

	for _, scope := range scopes {
		if scope == ScopeWrite {
			e.Rules[role][ScopeAdd] = true
			e.Rules[role][ScopeEdit] = true
			e.Rules[role][ScopeDelete] = true
		} else if scope == ScopeOwn {
			e.Rules[role][ScopeRead] = true
			e.Rules[role][ScopeAdd] = true
			e.Rules[role][ScopeEdit] = true
			e.Rules[role][ScopeDelete] = true
		}

		e.Rules[role][scope] = true
	}
}
