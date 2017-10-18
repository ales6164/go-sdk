package sdk

type Role map[string]map[Scope]bool

var userRoles = map[string]Role{}

func init() {
	SetUserRole("admin", "all", ScopeOwn)
	SetUserRole("client", "all", ScopeOwn)
}

func (c Context) HasScope(groupName string, scope Scope) bool {
	if c.scopes != nil {
		if s, ok := c.scopes[scope]; ok {
			return s
		}
	}

	if role, ok := userRoles[c.Role]; ok {
		if group, ok := role[groupName]; ok {
			if s, ok := group[scope]; ok {
				return s
			}
		} else if group, ok := role["all"]; ok {
			if s, ok := group[scope]; ok {
				return s
			}
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

func SetUserRole(roleName string, groupName string, scopes ...Scope) {
	if _, ok := userRoles[roleName]; !ok {
		userRoles[roleName] = Role{}
	}

	var role = map[Scope]bool{}
	for _, scope := range scopes {
		if scope == ScopeWrite {
			role[ScopeAdd] = true
			role[ScopeEdit] = true
			role[ScopeDelete] = true
		} else if scope == ScopeOwn {
			role[ScopeRead] = true
			role[ScopeAdd] = true
			role[ScopeEdit] = true
			role[ScopeDelete] = true
		}

		role[scope] = true
	}

	userRoles[roleName][groupName] = role
}
