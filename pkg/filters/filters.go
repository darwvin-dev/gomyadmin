package filters

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/darwvin-dev/gomyadmin/pkg/pagination"
)

type FieldType string

const (
	TypeString   FieldType = "string"
	TypeNumber   FieldType = "number"
	TypeBoolean  FieldType = "boolean"
	TypeEnum     FieldType = "enum"
	TypeDate     FieldType = "date"
	TypeDateTime FieldType = "datetime"
	TypeJSONB    FieldType = "jsonb"
	TypeRelation FieldType = "relation"
)

type AllowedField struct {
	Name       string
	Column     string
	Type       FieldType
	Searchable bool
	Sortable   bool
	Filterable bool
}

type Query struct {
	Page    pagination.Page `json:"page"`
	Search  string          `json:"search,omitempty"`
	Sorts   []Sort          `json:"sorts,omitempty"`
	Filters []Filter        `json:"filters,omitempty"`
}

type Sort struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc"`
}

type Filter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func Parse(values url.Values, allowed map[string]AllowedField) (Query, error) {
	query := Query{
		Page:   pagination.Parse(values),
		Search: strings.TrimSpace(values.Get("q")),
	}
	if query.Search == "" {
		query.Search = strings.TrimSpace(values.Get("search"))
	}

	sorts, err := parseSort(values.Get("sort"), allowed)
	if err != nil {
		return Query{}, err
	}
	query.Sorts = sorts

	filters, err := parseFilters(values, allowed)
	if err != nil {
		return Query{}, err
	}
	query.Filters = filters
	return query, nil
}

func parseSort(value string, allowed map[string]AllowedField) ([]Sort, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	sorts := make([]Sort, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		desc := strings.HasPrefix(part, "-")
		field := strings.TrimPrefix(part, "-")
		allowedField, ok := allowed[field]
		if !ok || !allowedField.Sortable {
			return nil, fmt.Errorf("field %q is not sortable", field)
		}
		sorts = append(sorts, Sort{Field: field, Desc: desc})
	}
	return sorts, nil
}

func parseFilters(values url.Values, allowed map[string]AllowedField) ([]Filter, error) {
	var out []Filter
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		if !strings.HasPrefix(key, "filter[") {
			continue
		}
		if len(value) == 0 || value[0] == "" {
			continue
		}

		field, op, ok := parseFilterKey(key)
		if !ok {
			return nil, fmt.Errorf("invalid filter key %q", key)
		}
		allowedField, exists := allowed[field]
		if !exists || !allowedField.Filterable {
			return nil, fmt.Errorf("field %q is not filterable", field)
		}
		if op == "" {
			op = defaultOperator(allowedField.Type)
		}
		if !operatorAllowed(allowedField.Type, op) {
			return nil, fmt.Errorf("operator %q is not allowed for %q", op, field)
		}
		out = append(out, Filter{Field: field, Operator: op, Value: value[0]})
	}
	return out, nil
}

func parseFilterKey(key string) (field string, operator string, ok bool) {
	if !strings.HasPrefix(key, "filter[") || !strings.HasSuffix(key, "]") {
		return "", "", false
	}
	trimmed := strings.TrimPrefix(key, "filter[")
	parts := strings.Split(trimmed, "][")
	if len(parts) == 0 {
		return "", "", false
	}
	field = strings.TrimSuffix(parts[0], "]")
	if field == "" {
		return "", "", false
	}
	if len(parts) > 1 {
		operator = strings.TrimSuffix(parts[1], "]")
	}
	return field, operator, true
}

func defaultOperator(fieldType FieldType) string {
	switch fieldType {
	case TypeString:
		return "contains"
	case TypeJSONB:
		return "contains"
	default:
		return "eq"
	}
}

func operatorAllowed(fieldType FieldType, operator string) bool {
	switch operator {
	case "eq":
		return true
	case "contains", "starts_with", "ends_with":
		return fieldType == TypeString || fieldType == TypeJSONB
	case "gte", "lte", "gt", "lt":
		return fieldType == TypeNumber || fieldType == TypeDate || fieldType == TypeDateTime
	case "in":
		return fieldType == TypeEnum || fieldType == TypeRelation
	default:
		return false
	}
}
