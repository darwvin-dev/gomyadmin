package openapi

import (
	"encoding/json"

	"github.com/darwvin/gomyadmin/pkg/admin"
)

type Spec struct {
	OpenAPI string          `json:"openapi"`
	Info    Info            `json:"info"`
	Paths   map[string]Path `json:"paths"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Path map[string]Operation

type Operation struct {
	Summary     string              `json:"summary"`
	OperationID string              `json:"operationId"`
	Tags        []string            `json:"tags"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	Schema      Schema `json:"schema"`
}

type RequestBody struct {
	Required bool               `json:"required,omitempty"`
	Content  map[string]Content `json:"content"`
}

type Content struct {
	Schema Schema `json:"schema"`
}

type Response struct {
	Description string             `json:"description"`
	Content     map[string]Content `json:"content,omitempty"`
}

type Schema struct {
	Type       string            `json:"type,omitempty"`
	Format     string            `json:"format,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
	Enum       []string          `json:"enum,omitempty"`
	Required   []string          `json:"required,omitempty"`
}

func Generate(app *admin.App, version string) Spec {
	spec := Spec{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:   app.Name() + " Admin API",
			Version: version,
		},
		Paths: map[string]Path{
			"/admin/api/resources": {
				"get": operation("List resources", "listResources", "Metadata"),
			},
			"/admin/api/me": {
				"get": operation("Current user", "getCurrentUser", "Auth"),
			},
			"/admin/api/audit": {
				"get": operation("List audit events", "listAuditEvents", "Audit"),
			},
		},
	}

	for _, resource := range app.Resources() {
		schema := resourceSchema(resource)
		base := "/admin/api/" + resource.Name
		spec.Paths[base] = Path{
			"get":  listOperation("List "+resource.PluralText, "list"+resource.Name, resource.LabelText),
			"post": writeOperation("Create "+resource.LabelText, "create"+resource.Name, resource.LabelText, schema),
		}
		spec.Paths[base+"/{id}"] = Path{
			"get":    detailOperation("Get "+resource.LabelText, "get"+resource.Name, resource.LabelText, schema),
			"patch":  writeOperation("Update "+resource.LabelText, "update"+resource.Name, resource.LabelText, partialSchema(resource)),
			"delete": deleteOperation("Delete "+resource.LabelText, "delete"+resource.Name, resource.LabelText),
		}
		for _, action := range resource.Actions {
			spec.Paths[base+"/{id}/actions/"+action.Name] = Path{
				"post": actionOperation(action.Label, resource.Name+"Action"+action.Name, resource.LabelText),
			}
		}
	}
	return spec
}

func JSON(spec Spec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

func operation(summary, id, tag string) Operation {
	return Operation{
		Summary:     summary,
		OperationID: id,
		Tags:        []string{tag},
		Responses: map[string]Response{
			"200": {Description: "OK"},
			"400": {Description: "Bad request"},
			"401": {Description: "Unauthenticated"},
			"403": {Description: "Forbidden"},
			"500": {Description: "Internal server error"},
		},
	}
}

func listOperation(summary, id, tag string) Operation {
	op := operation(summary, id, tag)
	op.Parameters = []Parameter{
		{Name: "page", In: "query", Description: "Page number", Schema: Schema{Type: "integer"}},
		{Name: "per_page", In: "query", Description: "Items per page", Schema: Schema{Type: "integer"}},
	}
	return op
}

func detailOperation(summary, id, tag string, schema Schema) Operation {
	op := operation(summary, id, tag)
	op.Parameters = []Parameter{
		{Name: "id", In: "path", Required: true, Description: "Resource identifier", Schema: Schema{Type: "string"}},
	}
	op.Responses["200"] = Response{
		Description: "OK",
		Content: map[string]Content{
			"application/json": {Schema: schema},
		},
	}
	return op
}

func deleteOperation(summary, id, tag string) Operation {
	op := operation(summary, id, tag)
	op.Parameters = []Parameter{
		{Name: "id", In: "path", Required: true, Description: "Resource identifier", Schema: Schema{Type: "string"}},
	}
	op.Responses["200"] = Response{Description: "Deleted"}
	return op
}

func writeOperation(summary, id, tag string, schema Schema) Operation {
	op := detailOperation(summary, id, tag, schema)
	op.RequestBody = &RequestBody{
		Required: true,
		Content: map[string]Content{
			"application/json": {Schema: schema},
		},
	}
	return op
}

func actionOperation(summary, id, tag string) Operation {
	op := operation(summary, id, tag)
	op.Parameters = []Parameter{
		{Name: "id", In: "path", Required: true, Description: "Resource identifier", Schema: Schema{Type: "string"}},
	}
	op.RequestBody = &RequestBody{
		Required: false,
		Content: map[string]Content{
			"application/json": {Schema: Schema{Type: "object"}},
		},
	}
	return op
}

func resourceSchema(resource *admin.Resource) Schema {
	properties := map[string]Schema{}
	required := make([]string, 0)
	for _, field := range resource.Fields {
		properties[field.Name] = fieldSchema(field)
		if field.RequiredValue {
			required = append(required, field.Name)
		}
	}
	return Schema{Type: "object", Properties: properties, Required: required}
}

func partialSchema(resource *admin.Resource) Schema {
	schema := resourceSchema(resource)
	schema.Required = nil
	return schema
}

func fieldSchema(field *admin.Field) Schema {
	switch field.Type {
	case admin.FieldBoolean:
		return Schema{Type: "boolean"}
	case admin.FieldInteger:
		return Schema{Type: "integer"}
	case admin.FieldNumber:
		return Schema{Type: "number"}
	case admin.FieldFloat, admin.FieldDecimal, admin.FieldMoney:
		return Schema{Type: "number"}
	case admin.FieldJSON, admin.FieldJSONB:
		return Schema{Type: "object"}
	case admin.FieldEnum, admin.FieldStatus:
		return Schema{Type: "string", Enum: field.EnumValues}
	case admin.FieldUUID:
		return Schema{Type: "string", Format: "uuid"}
	case admin.FieldDate:
		return Schema{Type: "string", Format: "date"}
	case admin.FieldDateTime:
		return Schema{Type: "string", Format: "date-time"}
	case admin.FieldRelation:
		return Schema{Type: "object"}
	case admin.FieldFile, admin.FieldImage:
		return Schema{Type: "string", Format: "uri"}
	default:
		return Schema{Type: "string"}
	}
}
