package admin

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// App holds the resource registry used by generators and runtime handlers.
type App struct {
	name      string
	mu        sync.RWMutex
	resources map[string]*Resource
}

// New creates an admin application registry.
func New(name string) *App {
	if strings.TrimSpace(name) == "" {
		name = "GoMyAdmin"
	}

	return &App{
		name:      name,
		resources: map[string]*Resource{},
	}
}

// Name returns the display name of the admin application.
func (a *App) Name() string {
	return a.name
}

// Resource registers or returns a resource for the supplied model value.
func (a *App) Resource(model any) *Resource {
	a.mu.Lock()
	defer a.mu.Unlock()
	name := resourceName(model)
	if existing, ok := a.resources[name]; ok {
		return existing
	}

	resource := &Resource{
		Name:            name,
		LabelText:       titleize(name),
		PluralText:      pluralize(titleize(name)),
		Table:           snakePlural(name),
		PrimaryKey:      "id",
		DefaultSort:     "-created_at",
		DefaultPageSize: 25,
		Fields:          []*Field{},
		Actions:         []*Action{},
	}
	a.resources[name] = resource
	return resource
}

// Register inserts a pre-built resource definition.
func (a *App) Register(resource *Resource) {
	if resource == nil || resource.Name == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resources[resource.Name] = resource
}

// GetResource returns a resource by name.
func (a *App) GetResource(name string) (*Resource, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	resource, ok := a.resources[name]
	return resource, ok
}

// Resources returns resources in stable name order.
func (a *App) Resources() []*Resource {
	a.mu.RLock()
	defer a.mu.RUnlock()
	names := make([]string, 0, len(a.resources))
	for name := range a.resources {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]*Resource, 0, len(names))
	for _, name := range names {
		out = append(out, a.resources[name])
	}
	return out
}

func resourceName(model any) string {
	if model == nil {
		return "resource"
	}

	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" {
		return "resource"
	}
	words := splitWords(t.Name())
	if len(words) == 0 {
		return "resource"
	}
	name := strings.ToLower(words[0])
	for _, word := range words[1:] {
		name += upperFirst(strings.ToLower(word))
	}
	return name
}

func titleize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	parts := splitWords(value)
	for i, part := range parts {
		if isAllUpper(part) {
			parts[i] = part
			continue
		}
		parts[i] = upperFirst(strings.ToLower(part))
	}
	return strings.Join(parts, " ")
}

func pluralize(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return value + "es"
	case strings.HasSuffix(lower, "y") && len(lower) > 1 && !isVowel(rune(lower[len(lower)-2])):
		return value[:len(value)-1] + "ies"
	default:
		return value + "s"
	}
}

func snakePlural(value string) string {
	parts := splitWords(value)
	for i, part := range parts {
		parts[i] = strings.ToLower(part)
	}
	return pluralize(strings.Join(parts, "_"))
}

func splitWords(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var words []string
	var current []rune
	runes := []rune(value)
	flush := func() {
		if len(current) == 0 {
			return
		}
		words = append(words, string(current))
		current = nil
	}
	for i, r := range runes {
		if r == '_' || r == '-' || unicode.IsSpace(r) {
			flush()
			continue
		}
		if len(current) > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower && len(current) > 1) {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return words
}

func upperFirst(value string) string {
	if value == "" {
		return value
	}
	r, size := utf8.DecodeRuneInString(value)
	return string(unicode.ToUpper(r)) + value[size:]
}

func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func isAllUpper(value string) bool {
	hasLetter := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}
