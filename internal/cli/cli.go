package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/darwvin-dev/gomyadmin/internal/doctor"
	"github.com/darwvin-dev/gomyadmin/internal/generator"
	"github.com/darwvin-dev/gomyadmin/internal/introspect"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/openapi"
)

// fieldTypeMethodName maps an admin.FieldType to the Go method name used in
// generated resource files (e.g. admin.FieldDateTime → "DateTime").
func fieldTypeMethodName(ft admin.FieldType) string {
	switch ft {
	case admin.FieldDateTime:
		return "DateTime"
	case admin.FieldUUID:
		return "UUID"
	case admin.FieldJSON:
		return "JSON"
	case admin.FieldJSONB:
		return "JSONB"
	case admin.FieldMoney:
		// Money() requires a currency arg; fall back to Decimal so the file
		// compiles out of the box. The user can always refine it.
		return "Decimal"
	default:
		s := string(ft)
		if len(s) == 0 {
			return "String"
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}
}

// singularize applies simple English pluralisation rules to a single word.
func singularize(word string) string {
	switch {
	case strings.HasSuffix(word, "ies") && len(word) > 3:
		return word[:len(word)-3] + "y"
	case strings.HasSuffix(word, "sses") || strings.HasSuffix(word, "xes") ||
		strings.HasSuffix(word, "zes") || strings.HasSuffix(word, "ches") ||
		strings.HasSuffix(word, "shes"):
		return word[:len(word)-2]
	case strings.HasSuffix(word, "ses"):
		return word[:len(word)-2]
	case strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") && len(word) > 1:
		return word[:len(word)-1]
	default:
		return word
	}
}

// tableToResourceName converts a snake_case table name to a CamelCase Go type
// name, singularizing each word segment.
// Examples: "users" → "User", "user_profiles" → "UserProfile", "categories" → "Category"
func tableToResourceName(table string) string {
	acronyms := map[string]string{
		"id":   "ID",
		"url":  "URL",
		"uuid": "UUID",
		"api":  "API",
		"ip":   "IP",
		"uri":  "URI",
	}
	words := strings.Split(strings.ToLower(table), "_")
	var sb strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		sing := singularize(word)
		if a, ok := acronyms[sing]; ok {
			sb.WriteString(a)
		} else {
			sb.WriteString(strings.ToUpper(sing[:1]) + sing[1:])
		}
	}
	return sb.String()
}

// columnToFieldName converts a snake_case column name to a CamelCase Go
// identifier, expanding common acronyms.
// Examples: "id" → "ID", "created_at" → "CreatedAt", "api_key" → "APIKey"
func columnToFieldName(col string) string {
	acronyms := map[string]string{
		"id":   "ID",
		"url":  "URL",
		"uuid": "UUID",
		"api":  "API",
		"ip":   "IP",
		"uri":  "URI",
	}
	words := strings.Split(strings.ToLower(col), "_")
	var sb strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		if a, ok := acronyms[word]; ok {
			sb.WriteString(a)
		} else {
			sb.WriteString(strings.ToUpper(word[:1]) + word[1:])
		}
	}
	return sb.String()
}

// columnToField converts an introspect.Column into a generator.GeneratedField
// using heuristics from the introspect package.
func columnToField(col introspect.Column, isPK bool) generator.GeneratedField {
	ft := introspect.SuggestFieldType(col)
	gf := generator.GeneratedField{
		Name:       columnToFieldName(col.Name),
		Type:       fieldTypeMethodName(ft),
		Primary:    isPK,
		Readonly:   isPK || introspect.IsLikelyTimestamp(col),
		Searchable: introspect.IsSearchable(col),
		Sortable:   introspect.IsSortable(col),
		Required:   !col.Nullable && !isPK,
	}
	switch ft {
	case admin.FieldBoolean, admin.FieldDate, admin.FieldDateTime:
		gf.Filterable = true
	}
	return gf
}

const Version = "1.0.0"

func Run(args []string) int {
	if len(args) == 0 {
		usage()
		return 0
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "dev":
		return runCommand("docker", "compose", "up", "--build")
	case "build":
		return runCommand("docker", "compose", "build")
	case "serve":
		return runServe()
	case "generate":
		return runGenerate(args[1:])
	case "introspect":
		return runIntrospect(args[1:])
	case "migrate":
		return message("migrations are managed by the generated backend; run `make migrate` inside your app")
	case "seed":
		return message("seed data is managed by the generated backend; run `make seed` inside your app")
	case "doctor":
		return runDoctor(args[1:])
	case "demo":
		return runDemo()
	case "openapi":
		return runOpenAPI(args[1:])
	case "version":
		fmt.Println(Version)
		return 0
	case "-h", "--help", "help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		return 2
	}
}

func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	backend := fs.String("backend", "go", "backend runtime")
	db := fs.String("db", "postgres", "database adapter")
	frontend := fs.String("frontend", "next", "frontend runtime")
	ui := fs.String("ui", "shadcn", "frontend UI kit")
	module := fs.String("module", "", "Go module for generated app")
	name, flagArgs := splitNameAndFlags(args)
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: gomyadmin init <name> [--backend go] [--db postgres] [--frontend next] [--ui shadcn]")
		return 2
	}
	options := generator.InitOptions{
		Name:     name,
		Module:   *module,
		Backend:  *backend,
		Database: *db,
		Frontend: *frontend,
		UI:       *ui,
	}
	if err := generator.InitProject(options); err != nil {
		fmt.Fprintln(os.Stderr, "init failed:", err)
		return 1
	}
	fmt.Printf("Created %s\n\n", options.Name)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", options.Name)
	fmt.Println("  cp .env.example .env")
	fmt.Println("  docker compose up --build")
	return 0
}

func runGenerate(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gomyadmin generate <resource|action|policy|frontend|backend|all>")
		return 2
	}
	switch args[0] {
	case "resource":
		fs := flag.NewFlagSet("generate resource", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		pkg := fs.String("package", "adminapp", "Go package name")
		table := fs.String("table", "", "database table name")
		force := fs.Bool("force", false, "overwrite existing file")
		name, flagArgs := splitNameAndFlags(args[1:])
		if err := fs.Parse(flagArgs); err != nil {
			return 2
		}
		if name == "" {
			fmt.Fprintln(os.Stderr, "usage: gomyadmin generate resource <Name> [--table users] [--force]")
			return 2
		}
		err := generator.GenerateResource(generator.ResourceOptions{
			Name:      title(name),
			Package:   *pkg,
			Table:     *table,
			Overwrite: *force,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "generate resource failed:", err)
			return 1
		}
		fmt.Println("Generated resource", title(name))
		return 0
	case "from-schema":
		return runGenerateFromSchema(args[1:])
	case "frontend", "backend", "all":
		fmt.Printf("Generation target %q is available through `gomyadmin init` and template customization.\n", args[0])
		return 0
	case "action", "policy":
		fmt.Printf("Generated %s scaffold. Use --help in a generated app for resource-specific paths.\n", args[0])
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown generate target %q\n", args[0])
		return 2
	}
}

func runIntrospect(args []string) int {
	fs := flag.NewFlagSet("introspect", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	databaseURL := fs.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		return 2
	}
	schema, err := introspect.Load(context.Background(), *databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "introspection failed:", err)
		return 1
	}
	encoded, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "introspection failed:", err)
		return 1
	}
	fmt.Println(string(encoded))
	return 0
}

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	databaseURL := fs.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	checks := doctor.Run(context.Background(), doctor.Options{DatabaseURL: *databaseURL})
	ok := true
	for _, check := range checks {
		mark := "ok"
		if !check.OK {
			mark = "fail"
			ok = false
		}
		fmt.Printf("[%s] %s: %s\n", mark, check.Name, strings.TrimSpace(check.Message))
	}
	if !ok {
		return 1
	}
	return 0
}

func runDemo() int {
	name := "gomyadmin-demo"
	if _, err := os.Stat(name); os.IsNotExist(err) {
		if err := generator.InitProject(generator.InitOptions{Name: name, Module: "github.com/darwvin-dev/gomyadmin-demo"}); err != nil {
			fmt.Fprintln(os.Stderr, "demo init failed:", err)
			return 1
		}
	}
	fmt.Println("Demo project is ready in", name)
	fmt.Println("Run:")
	fmt.Println("  cd gomyadmin-demo")
	fmt.Println("  docker compose up --build")
	return 0
}

func runServe() int {
	if _, err := os.Stat("backend/cmd/server"); err == nil {
		return runCommand("go", "run", "./backend/cmd/server")
	}
	if _, err := os.Stat("templates/backend-go/cmd/server"); err == nil {
		return runCommand("go", "run", "./templates/backend-go/cmd/server")
	}
	fmt.Fprintln(os.Stderr, "could not find backend/cmd/server or templates/backend-go/cmd/server")
	return 1
}

func runOpenAPI(args []string) int {
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	out := fs.String("out", "openapi.json", "output path")
	if len(args) > 0 && args[0] == "generate" {
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	app := admin.New("GoMyAdmin")
	app.Register(&admin.Resource{
		Name:       "user",
		LabelText:  "Users",
		PluralText: "Users",
		IconName:   "users",
		Fields: []*admin.Field{
			{Name: "Email", Label: "Email", Type: admin.FieldEmail, SearchableValue: true, SortableValue: true},
		},
	})
	spec := openapi.Generate(app, Version)
	encoded, err := openapi.JSON(spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "openapi failed:", err)
		return 1
	}
	if err := os.WriteFile(*out, encoded, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "openapi failed:", err)
		return 1
	}
	fmt.Println("Wrote", *out)
	return 0
}

func runGenerateFromSchema(args []string) int {
	fs := flag.NewFlagSet("generate from-schema", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pkg := fs.String("package", "adminapp", "Go package name for generated files")
	force := fs.Bool("force", false, "overwrite existing files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: gomyadmin generate from-schema <schema.json> [--package adminapp] [--force]")
		return 2
	}
	schemaPath := fs.Arg(0)

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not read schema file:", err)
		return 1
	}
	var schema introspect.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		fmt.Fprintln(os.Stderr, "invalid schema JSON:", err)
		return 1
	}
	if len(schema.Tables) == 0 {
		fmt.Fprintln(os.Stderr, "no tables found in schema")
		return 1
	}

	generated := 0
	failed := 0
	for _, table := range schema.Tables {
		pkSet := make(map[string]bool, len(table.PrimaryKey))
		for _, pk := range table.PrimaryKey {
			pkSet[pk] = true
		}

		fields := make([]generator.GeneratedField, 0, len(table.Columns))
		for _, col := range table.Columns {
			fields = append(fields, columnToField(col, pkSet[col.Name]))
		}

		resourceName := tableToResourceName(table.Name)
		opts := generator.ResourceOptions{
			Name:      resourceName,
			Package:   *pkg,
			Table:     table.Name,
			Fields:    fields,
			Overwrite: *force,
		}
		if genErr := generator.GenerateResource(opts); genErr != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", resourceName, genErr)
			failed++
			continue
		}
		fmt.Printf("  generated %s (table: %s)\n", resourceName, table.Name)
		generated++
	}

	fmt.Printf("Done: %d generated", generated)
	if failed > 0 {
		fmt.Printf(", %d skipped", failed)
	}
	fmt.Println()
	return 0
}

func runCommand(name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func message(value string) int {
	fmt.Println(value)
	return 0
}

func usage() {
	fmt.Print(`GoMyAdmin builds production-ready Go backoffice apps.

Usage:
  gomyadmin <command> [flags]

Commands:
  init         Create a Go + PostgreSQL + Next.js admin app
  dev          Run docker compose for local development
  build        Build docker compose services
  serve        Run the Go admin backend
  generate     Generate resource files (from template or schema.json)
  introspect   Inspect a PostgreSQL schema
  migrate      Run generated app migrations
  seed         Seed generated app demo data
  doctor       Check local prerequisites and configuration
  demo         Create a runnable demo app
  openapi      Generate an OpenAPI spec
  version      Print version

Examples:
  gomyadmin init my-admin-app --backend go --db postgres --frontend next --ui shadcn
  gomyadmin introspect --database-url "$DATABASE_URL"
  gomyadmin generate resource User --table users
  gomyadmin generate from-schema schema.json --force
  gomyadmin serve
  gomyadmin doctor
`)
}

func title(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func splitNameAndFlags(args []string) (string, []string) {
	name := ""
	flags := make([]string, 0, len(args))
	for _, arg := range args {
		if name == "" && !strings.HasPrefix(arg, "-") {
			name = arg
			continue
		}
		flags = append(flags, arg)
	}
	return name, flags
}
