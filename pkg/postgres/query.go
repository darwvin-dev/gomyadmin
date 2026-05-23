package postgres

import (
	"fmt"
	"strings"

	"github.com/darwvin/gomyadmin/pkg/filters"
)

type QueryBuilder struct {
	Table        string
	PrimaryKey   string
	Allowed      map[string]filters.AllowedField
	TenantColumn string
}

type BuiltQuery struct {
	SQL  string
	Args []any
}

func (b QueryBuilder) Select(query filters.Query, tenantID string) (BuiltQuery, error) {
	if err := validateIdentifier(b.Table); err != nil {
		return BuiltQuery{}, err
	}
	where, args, err := b.where(query, tenantID)
	if err != nil {
		return BuiltQuery{}, err
	}

	order, err := b.orderBy(query)
	if err != nil {
		return BuiltQuery{}, err
	}
	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, query.Page.Limit, query.Page.Offset)

	sql := fmt.Sprintf("select * from %s%s%s limit $%d offset $%d", quoteIdent(b.Table), where, order, limitArg, offsetArg)
	return BuiltQuery{SQL: sql, Args: args}, nil
}

func (b QueryBuilder) Count(query filters.Query, tenantID string) (BuiltQuery, error) {
	if err := validateIdentifier(b.Table); err != nil {
		return BuiltQuery{}, err
	}
	where, args, err := b.where(query, tenantID)
	if err != nil {
		return BuiltQuery{}, err
	}
	sql := fmt.Sprintf("select count(*) from %s%s", quoteIdent(b.Table), where)
	return BuiltQuery{SQL: sql, Args: args}, nil
}

func (b QueryBuilder) where(query filters.Query, tenantID string) (string, []any, error) {
	clauses := []string{}
	args := []any{}

	if b.TenantColumn != "" {
		if err := validateIdentifier(b.TenantColumn); err != nil {
			return "", nil, err
		}
		args = append(args, tenantID)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", quoteIdent(b.TenantColumn), len(args)))
	}

	if query.Search != "" {
		searchParts := []string{}
		for _, field := range b.Allowed {
			if !field.Searchable {
				continue
			}
			column := field.Column
			if column == "" {
				column = field.Name
			}
			if err := validateIdentifier(column); err != nil {
				return "", nil, err
			}
			args = append(args, "%"+query.Search+"%")
			searchParts = append(searchParts, fmt.Sprintf("%s ilike $%d", quoteIdent(column), len(args)))
		}
		if len(searchParts) > 0 {
			clauses = append(clauses, "("+strings.Join(searchParts, " or ")+")")
		}
	}

	for _, filter := range query.Filters {
		field, ok := b.Allowed[filter.Field]
		if !ok || !field.Filterable {
			return "", nil, fmt.Errorf("field %q is not filterable", filter.Field)
		}
		column := field.Column
		if column == "" {
			column = field.Name
		}
		if err := validateIdentifier(column); err != nil {
			return "", nil, err
		}
		args = append(args, filterArg(filter))
		clauses = append(clauses, fmt.Sprintf("%s %s $%d", quoteIdent(column), sqlOperator(filter.Operator), len(args)))
	}

	if len(clauses) == 0 {
		return "", args, nil
	}
	return " where " + strings.Join(clauses, " and "), args, nil
}

func (b QueryBuilder) orderBy(query filters.Query) (string, error) {
	orders := make([]string, 0, len(query.Sorts))
	for _, sort := range query.Sorts {
		field, ok := b.Allowed[sort.Field]
		if !ok || !field.Sortable {
			return "", fmt.Errorf("field %q is not sortable", sort.Field)
		}
		column := field.Column
		if column == "" {
			column = field.Name
		}
		if err := validateIdentifier(column); err != nil {
			return "", err
		}
		direction := "asc"
		if sort.Desc {
			direction = "desc"
		}
		orders = append(orders, quoteIdent(column)+" "+direction)
	}
	if b.PrimaryKey != "" {
		if err := validateIdentifier(b.PrimaryKey); err == nil {
			pk := quoteIdent(b.PrimaryKey) + " asc"
			found := false
			for _, order := range orders {
				if strings.HasPrefix(order, quoteIdent(b.PrimaryKey)+" ") {
					found = true
					break
				}
			}
			if !found {
				orders = append(orders, pk)
			}
		}
	}
	if len(orders) == 0 {
		return "", nil
	}
	return " order by " + strings.Join(orders, ", "), nil
}

func filterArg(filter filters.Filter) any {
	switch filter.Operator {
	case "contains":
		return "%" + filter.Value + "%"
	case "starts_with":
		return filter.Value + "%"
	case "ends_with":
		return "%" + filter.Value
	default:
		return filter.Value
	}
}

func sqlOperator(operator string) string {
	switch operator {
	case "contains", "starts_with", "ends_with":
		return "ilike"
	case "gte":
		return ">="
	case "lte":
		return "<="
	case "gt":
		return ">"
	case "lt":
		return "<"
	default:
		return "="
	}
}

func validateIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier is empty")
	}
	for _, r := range identifier {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return fmt.Errorf("unsafe identifier %q", identifier)
	}
	return nil
}

func quoteIdent(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
