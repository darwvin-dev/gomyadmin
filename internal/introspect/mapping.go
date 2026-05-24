package introspect

import (
	"strings"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

// ColumnToFieldType maps a PostgreSQL data_type string to the closest admin.FieldType.
// The mapping is intentionally conservative: types that need domain knowledge
// (e.g. an "integer" that represents money) should be overridden after generation.
func ColumnToFieldType(dataType string) admin.FieldType {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "uuid":
		return admin.FieldUUID
	case "boolean", "bool":
		return admin.FieldBoolean
	case "smallint", "integer", "int", "int2", "int4":
		return admin.FieldInteger
	case "bigint", "int8":
		return admin.FieldInteger
	case "real", "float4":
		return admin.FieldFloat
	case "double precision", "float8":
		return admin.FieldFloat
	case "numeric", "decimal":
		return admin.FieldDecimal
	case "money":
		return admin.FieldMoney
	case "text":
		return admin.FieldText
	case "character varying", "varchar", "character", "char", "bpchar":
		return admin.FieldString
	case "date":
		return admin.FieldDate
	case "time", "time without time zone", "time with time zone", "timetz":
		return admin.FieldTime
	case "timestamp", "timestamp without time zone":
		return admin.FieldDateTime
	case "timestamp with time zone", "timestamptz":
		return admin.FieldDateTime
	case "json":
		return admin.FieldJSON
	case "jsonb":
		return admin.FieldJSONB
	default:
		return admin.FieldString
	}
}

// IsLikelyPrimaryKey returns true when the column name and type look like a primary key.
func IsLikelyPrimaryKey(col Column) bool {
	name := strings.ToLower(col.Name)
	return name == "id" || col.IsIdentity ||
		(name == "uuid" && col.DataType == "uuid") ||
		strings.HasSuffix(name, "_id") && col.IsIdentity
}

// IsLikelyTimestamp returns true when a column appears to be a timestamp field.
func IsLikelyTimestamp(col Column) bool {
	lower := strings.ToLower(col.DataType)
	return strings.Contains(lower, "timestamp") || strings.Contains(lower, "timetz")
}

// IsLikelyEmail returns true when a column name strongly suggests an email address.
func IsLikelyEmail(col Column) bool {
	name := strings.ToLower(col.Name)
	return name == "email" || strings.HasSuffix(name, "_email")
}

// IsLikelyPassword returns true when a column name strongly suggests a stored password.
func IsLikelyPassword(col Column) bool {
	name := strings.ToLower(col.Name)
	return name == "password" || name == "password_hash" || name == "hashed_password" || strings.HasSuffix(name, "_password")
}

// SuggestFieldType returns a more precise FieldType by considering both the
// PostgreSQL data type and the column name heuristics.
func SuggestFieldType(col Column) admin.FieldType {
	if IsLikelyEmail(col) {
		return admin.FieldEmail
	}
	if IsLikelyPassword(col) {
		return admin.FieldPassword
	}
	return ColumnToFieldType(col.DataType)
}

// IsSearchable returns true for column types that are reasonable to full-text search.
func IsSearchable(col Column) bool {
	switch ColumnToFieldType(col.DataType) {
	case admin.FieldString, admin.FieldText, admin.FieldEmail:
		return true
	default:
		return IsLikelyEmail(col)
	}
}

// IsSortable returns true for column types that support ORDER BY sensibly.
func IsSortable(col Column) bool {
	switch ColumnToFieldType(col.DataType) {
	case admin.FieldString, admin.FieldText, admin.FieldEmail,
		admin.FieldInteger, admin.FieldFloat, admin.FieldDecimal, admin.FieldMoney,
		admin.FieldDate, admin.FieldDateTime, admin.FieldTime,
		admin.FieldBoolean:
		return true
	default:
		return false
	}
}
