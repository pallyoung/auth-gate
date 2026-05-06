package router

import "github.com/pallyoung/auth-gate/packages/server/internal/store"

func compileRoutes(routes []store.Route, authRules []store.AuthRule) []Route {
	authMap := make(map[string]AuthRule, len(authRules))
	for _, rule := range authRules {
		authMap[rule.RouteID] = compileAuthRule(rule)
	}

	result := make([]Route, 0, len(routes))
	for _, route := range routes {
		compiledRoute := Route{
			ID:          route.ID,
			Name:        route.Name,
			Host:        route.Host,
			PathPrefix:  route.PathPrefix,
			Backend:     route.Backend,
			StripPrefix: route.StripPrefix,
			Enabled:     route.Enabled,
			Priority:    route.Priority,
		}
		if rule, ok := authMap[route.ID]; ok {
			ruleCopy := rule
			compiledRoute.AuthRule = &ruleCopy
		}
		result = append(result, compiledRoute)
	}
	return result
}

func compileAuthRule(rule store.AuthRule) AuthRule {
	return AuthRule{
		ID:      rule.ID,
		RouteID: rule.RouteID,
		Type:    rule.Type,
		Config: AuthConfig{
			HeaderName: rule.Config.HeaderName,
			Secret:     rule.Config.Secret,
			Username:   rule.Config.Username,
			Password:   rule.Config.Password,
		},
	}
}
