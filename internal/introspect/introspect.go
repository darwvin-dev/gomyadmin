package introspect

// Schema holds the introspected PostgreSQL schema.
type Schema struct {
	Tables []Table `json:"tables"`
}

// Table represents a single database table.
type Table struct {
	Schema     string   `json:"schema"`
	Name       string   `json:"name"`
	Columns    []Column `json:"columns"`
	PrimaryKey []string `json:"primary_key"`
}

// Column represents a single column in a table.
type Column struct {
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	Nullable   bool   `json:"nullable"`
	Default    string `json:"default,omitempty"`
	IsIdentity bool   `json:"is_identity"`
}
