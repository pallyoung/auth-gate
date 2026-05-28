package router

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/pallyoung/auth-gate/packages/server/internal/routehost"
	"github.com/pallyoung/auth-gate/packages/server/internal/store"
)

// Match modes ordered by nginx priority (highest to lowest):
//   =        exact match   (priority 100)
//   ^~       prefix stop  (priority 90)
//   ~ / ~*   regex        (priority 80)
//   (plain)  prefix       (priority 70)

const (
	matchPriorityExact  = 100 // "=" exact match
	matchPriorityStop   = 90  // "^~" regex-prefix stop
	matchPriorityRegex  = 80  // "~" or "~*" regex
	matchPriorityPrefix = 70  // plain prefix
)

// parsePathMatchMode parses nginx-style path prefix syntax.
// Supported patterns:
//
//	=  /exact/path    → exact match
//	~  ~/regex$       → case-sensitive regex
//	~* ~/~*[Rr]egex$  → case-insensitive regex
//	^~ ~/prefix       → stop on first prefix match
//	(none) /prefix    → plain prefix match
func parsePathMatchMode(pathPrefix string) (mode string, pattern string) {
	s := strings.TrimLeft(pathPrefix, " \t")
	if s == "" {
		return "prefix", pathPrefix
	}
	if len(s) >= 2 {
		switch s[:2] {
		case "= ":
			return "exact", strings.TrimLeft(s[2:], " ")
		case "^~ ":
			return "stop", strings.TrimLeft(s[2:], " ")
		}
	}
	if s[0] == '~' {
		if len(s) >= 3 && (s[1] == ' ' || s[1] == '*') {
			if s[1] == '*' {
				if len(s) >= 3 && s[2] == ' ' {
					return "regex_i", strings.TrimLeft(s[3:], " ")
				}
			} else if len(s) >= 2 && s[1] == ' ' {
				return "regex", strings.TrimLeft(s[2:], " ")
			}
		}
	}
	return "prefix", pathPrefix
}

func matchModeDisplay(mode, pathPrefix string) string {
	switch mode {
	case "exact":
		return "= " + pathPrefix
	case "stop":
		return "^~ " + pathPrefix
	case "regex":
		return "~ " + pathPrefix
	case "regex_i":
		return "~* " + pathPrefix
	default:
		return pathPrefix
	}
}

func matchPriority(mode string) int {
	switch mode {
	case "exact":
		return matchPriorityExact
	case "stop":
		return matchPriorityStop
	case "regex", "regex_i":
		return matchPriorityRegex
	default:
		return matchPriorityPrefix
	}
}

func compileRoutes(routes []store.Route, authRules []store.AuthRule) []Route {
	authMap := make(map[string]AuthRule, len(authRules))
	for _, rule := range authRules {
		authMap[rule.RouteID] = compileAuthRule(rule)
	}

	compiled := make([]Route, 0, len(routes))
	for _, route := range routes {
		effectiveMode, ok := normalizeStoredPathMatchMode(route.PathMatchMode)
		if !ok {
			// Skip malformed legacy rows so unsupported stored modes never capture traffic.
			continue
		}

		compiledRoute := Route{
			ID:            route.ID,
			Name:          route.Name,
			Host:          normalizeStoredHost(route.Host),
			Backend:       route.Backend,
			Backends:      route.Backends,
			StripPrefix:   route.StripPrefix,
			Enabled:       route.Enabled,
			Priority:      route.Priority,
			TimeoutMs:     route.TimeoutMs,
			RetryAttempts: route.RetryAttempts,
			PathMatchMode: route.PathMatchMode,
			RewriteTarget: strings.TrimSpace(route.RewriteTarget),
			RedirectCode:  normalizeStoredRedirectCode(route.RedirectCode),
		}

		if effectiveMode == "" {
			effectiveMode, compiledRoute.PathPrefix = parsePathMatchMode(route.PathPrefix)
		} else {
			compiledRoute.PathPrefix = route.PathPrefix
		}
		compiledRoute.PathMatchMode = effectiveMode

		basePriority := matchPriority(effectiveMode)
		compiledRoute.Priority = basePriority + route.Priority

		if effectiveMode == "regex" || effectiveMode == "regex_i" {
			pattern := compiledRoute.PathPrefix
			if effectiveMode == "regex_i" {
				pattern = "(?i)" + pattern
			}
			re, err := regexp.Compile(pattern)
			if err == nil {
				compiledRoute.PathRegex = re
			}
		}

		if rule, ok := authMap[route.ID]; ok {
			ruleCopy := rule
			compiledRoute.AuthRule = &ruleCopy
		}
		compiled = append(compiled, compiledRoute)
	}

	sortRoutes(compiled)
	return compiled
}

// sortRoutes sorts routes so Match() returns the correct nginx priority result.
// Within the same match-order class, longer path patterns come first (longest-prefix wins).
// sortByPriority uses priorityCmp: returns > 0 when a should come BEFORE b.
func sortRoutes(routes []Route) {
	for i := range routes {
		if routes[i].Priority == 0 {
			routes[i].Priority = matchPriority(routes[i].PathMatchMode)
		}
	}
	sortByPriority(routes)
}

// sortByPriority: if priorityCmp(a, b) < 0, swap(a, b).
// priorityCmp < 0 means b has higher priority and should be earlier.
func sortByPriority(routes []Route) {
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if priorityCmp(routes[i], routes[j]) < 0 {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}
}

// priorityCmp: positive → a has higher priority; negative → b has higher priority.
// Higher priority = evaluated later in Match = wins.
// nginx evaluation order: exact(4) > stop(3) > regex(2) > prefix(1).
// Within same class: longer path pattern wins.
func priorityCmp(a, b Route) int {
	ao := matchOrder(a)
	bo := matchOrder(b)
	if ao != bo {
		return ao - bo // positive = a higher order = a wins (evaluated later)
	}
	return pathLenCmp(a, b) // positive = a longer = a wins
}

// matchOrder returns the sort order for a route (higher = evaluated later in Match).
func matchOrder(r Route) int {
	switch r.PathMatchMode {
	case "exact":
		return 4
	case "stop":
		return 3
	case "regex", "regex_i":
		return 2
	default:
		return 1
	}
}

// pathLenCmp: positive → a has longer path → a should win.
func pathLenCmp(a, b Route) int {
	la := len(a.PathPrefix)
	lb := len(b.PathPrefix)
	if la != lb {
		return la - lb // positive = a longer = a wins
	}
	if ap := a.Priority; ap != b.Priority {
		return ap - b.Priority // positive = a higher priority = a wins
	}
	// Stable tie-break: lexicographically greater ID wins (comes first).
	if a.ID > b.ID {
		return 1
	}
	if a.ID < b.ID {
		return -1
	}
	return 0
}

func compileAuthRule(rule store.AuthRule) AuthRule {
	return AuthRule{
		ID:      rule.ID,
		RouteID: rule.RouteID,
		Type:    normalizeStoredAuthRuleType(rule.Type),
		Config: AuthConfig{
			HeaderName: rule.Config.HeaderName,
			Secret:     rule.Config.Secret,
			Username:   rule.Config.Username,
			Password:   rule.Config.Password,
			LoginMode:  rule.Config.LoginMode,
		},
		Whitelist:            rule.Whitelist,
		RateLimit:            rule.RateLimit,
		Burst:                rule.Burst,
		CORSAllowedOrigins:   rule.CORSAllowedOrigins,
		CORSAllowedMethods:   rule.CORSAllowedMethods,
		CORSAllowedHeaders:   rule.CORSAllowedHeaders,
		CORSAllowCredentials: rule.CORSAllowCredentials,
		CORSMaxAge:           rule.CORSMaxAge,
	}
}

func normalizeStoredAuthRuleType(ruleType string) string {
	return strings.ToLower(strings.TrimSpace(ruleType))
}

func normalizeStoredRedirectCode(code int) int {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound:
		return code
	default:
		return 0
	}
}

func normalizeStoredPathMatchMode(pathMatchMode string) (string, bool) {
	pathMatchMode = strings.ToLower(strings.TrimSpace(pathMatchMode))
	switch pathMatchMode {
	case "", "prefix":
		return "", true
	case "exact", "stop", "regex", "regex_i":
		return pathMatchMode, true
	default:
		return "", false
	}
}

func normalizeStoredHost(host string) string {
	return routehost.Normalize(host)
}

// ApplyRewrite applies rewrite_target using regex capture groups.
func ApplyRewrite(path, rewriteTarget string, groups []string) string {
	if rewriteTarget == "" {
		return path
	}
	result := rewriteTarget
	result = strings.ReplaceAll(result, "$&", path)
	for i := len(groups); i >= 1; i-- {
		placeholder := "$" + string(rune('0'+i))
		if i <= len(groups) {
			result = strings.ReplaceAll(result, placeholder, groups[i-1])
		}
	}
	return result
}
