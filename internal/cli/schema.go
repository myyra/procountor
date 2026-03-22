package cli

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v3"

	"github.com/myyra/procountor/internal/apispec"
)

type SchemaCmd struct {
	Command []string `arg:"" help:"Command path to describe, for example: invoices save." name:"command" optional:""`
}

type schemaFlag struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Short       string   `json:"short,omitempty"`
	Help        string   `json:"help,omitempty"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	Default     string   `json:"default,omitempty"`
	HasDefault  bool     `json:"has_default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Envs        []string `json:"envs,omitempty"`
	Hidden      bool     `json:"hidden,omitempty"`
	Negated     bool     `json:"negated,omitempty"`
}

func (c *SchemaCmd) Run(_ context.Context, r *runner) error {
	parser, _, err := newParser(r, true)
	if err != nil {
		return wrapErr("build schema parser", err)
	}

	commandPath := splitCommandPath(c.Command)
	if len(commandPath) == 0 {
		return nil
	}

	node, err := findCommandNode(parser.Model.Node, commandPath)
	if err != nil {
		return err
	}

	componentName, ok := requestBodySchemaComponentName(node)
	if !ok {
		return nil
	}

	doc, err := requestBodySchemaDocument(componentName)
	if err != nil {
		return err
	}
	return r.writeJSONValue(doc)
}

var requestBodySchemaComponents = map[string]string{
	"attachments rename":                            "AttachmentRename",
	"business-partners patch":                       "BusinessPartner",
	"business-partners put":                         "BusinessPartner",
	"business-partners groups create":               "BusinessPartnerGroup",
	"business-partners groups update":               "BusinessPartnerGroup",
	"cost-receipts create":                          "CostReceipt",
	"dimensions add-item":                           "DimensionItem",
	"dimensions update":                             "DimensionName",
	"dimensions update-item":                        "DimensionItem",
	"invoices add-notes":                            "InvoiceNotes",
	"invoices approve":                              "CheckingEvent",
	"invoices comments create":                      "Comment",
	"invoices payment-events mark-paid":             "MarkInvoiceAsPaid",
	"invoices reject":                               "CheckingEvent",
	"invoices save":                                 "Invoice",
	"invoices verify":                               "CheckingEvent",
	"ledger-receipts create":                        "LedgerReceipt",
	"ledger-receipts update":                        "LedgerReceipt",
	"ledger-receipts update-transaction-dimensions": "DimensionItemValueList",
	"persons update":                                "Person",
	"reports accounting":                            "AccountingReportRequest",
	"reports general-ledger":                        "GeneralLedgerReportRequest",
	"reports ledger-accounts":                       "LedgerAccountsReportRequest",
}

type openAPISpec struct {
	Components struct {
		Schemas map[string]any `yaml:"schemas"`
	} `yaml:"components"`
}

var (
	schemaSpecOnce sync.Once
	schemaSpecDoc  openAPISpec
	errSchemaSpec  error
)

func requestBodySchemaComponentName(node *kong.Node) (string, bool) {
	componentName, ok := requestBodySchemaComponents[commandPathKey(node)]
	return componentName, ok
}

func requestBodySchemaDocument(componentName string) (map[string]any, error) {
	spec, err := loadSchemaSpec()
	if err != nil {
		return nil, err
	}

	transformer := openAPISchemaTransformer{
		components: spec.Components.Schemas,
		defs:       make(map[string]any),
		processing: make(map[string]bool),
	}
	if err := transformer.ensureDefinition(componentName); err != nil {
		return nil, err
	}

	doc := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$ref":    "#/$defs/" + componentName,
		"$defs":   transformer.defs,
	}
	return doc, nil
}

func loadSchemaSpec() (*openAPISpec, error) {
	schemaSpecOnce.Do(func() {
		errSchemaSpec = yaml.Unmarshal(apispec.ProcountorAPI, &schemaSpecDoc)
	})
	if errSchemaSpec != nil {
		return nil, wrapErr("parse OpenAPI spec", errSchemaSpec)
	}
	return &schemaSpecDoc, nil
}

func commandPathKey(node *kong.Node) string {
	if node == nil {
		return ""
	}
	parts := strings.Fields(node.FullPath())
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[1:], " ")
}

type openAPISchemaTransformer struct {
	components map[string]any
	defs       map[string]any
	processing map[string]bool
}

func (t *openAPISchemaTransformer) ensureDefinition(name string) error {
	if _, ok := t.defs[name]; ok {
		return nil
	}
	if t.processing[name] {
		return nil
	}

	raw, ok := t.components[name]
	if !ok {
		return fmt.Errorf("OpenAPI schema %q not found", name)
	}

	t.processing[name] = true
	defer delete(t.processing, name)

	transformed, err := t.transform(raw)
	if err != nil {
		return err
	}
	t.defs[name] = transformed
	return nil
}

func (t *openAPISchemaTransformer) transform(value any) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		if ref, ok := typed["$ref"].(string); ok {
			name, ok := schemaRefName(ref)
			if !ok {
				return nil, fmt.Errorf("unsupported schema ref %q", ref)
			}
			if err := t.ensureDefinition(name); err != nil {
				return nil, err
			}
			return map[string]any{"$ref": "#/$defs/" + name}, nil
		}

		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if key == "nullable" {
				continue
			}
			transformed, err := t.transform(item)
			if err != nil {
				return nil, err
			}
			out[key] = transformed
		}
		if nullable, _ := typed["nullable"].(bool); nullable {
			out = applyNullable(out)
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			transformed, err := t.transform(item)
			if err != nil {
				return nil, err
			}
			out = append(out, transformed)
		}
		return out, nil
	default:
		return value, nil
	}
}

func schemaRefName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return name, true
}

func applyNullable(schema map[string]any) map[string]any {
	if rawType, ok := schema["type"]; ok {
		switch typed := rawType.(type) {
		case string:
			schema["type"] = []any{typed, "null"}
			return schema
		case []any:
			if !containsAnyString(typed, "null") {
				schema["type"] = append(typed, "null")
			}
			return schema
		}
	}

	if rawAnyOf, ok := schema["anyOf"].([]any); ok {
		if !containsNullSchema(rawAnyOf) {
			schema["anyOf"] = append(rawAnyOf, map[string]any{"type": "null"})
		}
		return schema
	}

	if rawOneOf, ok := schema["oneOf"].([]any); ok {
		if !containsNullSchema(rawOneOf) {
			schema["oneOf"] = append(rawOneOf, map[string]any{"type": "null"})
		}
		return schema
	}

	return map[string]any{
		"anyOf": []any{
			schema,
			map[string]any{"type": "null"},
		},
	}
}

func containsAnyString(values []any, want string) bool {
	for _, value := range values {
		if s, ok := value.(string); ok && s == want {
			return true
		}
	}
	return false
}

func containsNullSchema(values []any) bool {
	for _, value := range values {
		schema, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if schemaType, ok := schema["type"].(string); ok && schemaType == scalarInputNullLiteral {
			return true
		}
	}
	return false
}

func splitCommandPath(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, token := range strings.Fields(part) {
			if token != "" {
				out = append(out, token)
			}
		}
	}
	return out
}

func findCommandNode(root *kong.Node, path []string) (*kong.Node, error) {
	current := root
	for _, token := range path {
		next := findChildCommand(current, token)
		if next == nil {
			return nil, invalidUsage("unknown command %q under %q", token, current.FullPath())
		}
		current = next
	}
	return current, nil
}

func findChildCommand(parent *kong.Node, token string) *kong.Node {
	want := strings.ToLower(token)
	for _, child := range parent.Children {
		if child == nil || child.Type != kong.CommandNode {
			continue
		}
		if strings.ToLower(child.Name) == want {
			return child
		}
		for _, alias := range child.Aliases {
			if strings.ToLower(alias) == want {
				return child
			}
		}
	}
	return nil
}

func schemaFlags(node *kong.Node, hide bool) []schemaFlag {
	var out []schemaFlag
	for _, group := range node.AllFlags(hide) {
		for _, flag := range group {
			if flag == nil {
				continue
			}
			out = append(out, schemaFlag{
				Name:        flag.Name,
				Aliases:     sortedStrings(flag.Aliases),
				Short:       flagShortString(flag.Short),
				Help:        flag.Help,
				Type:        reflectTypeString(flag.Target),
				Required:    flag.Required,
				Default:     flag.Default,
				HasDefault:  flag.HasDefault,
				Enum:        sortedStrings(flag.EnumSlice()),
				Placeholder: flag.FormatPlaceHolder(),
				Envs:        sortedStrings(flag.Envs),
				Hidden:      flag.Hidden,
				Negated:     flag.Negated,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func reflectTypeString(v any) string {
	if v == nil {
		return ""
	}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	return t.String()
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func flagShortString(short rune) string {
	if short == 0 {
		return ""
	}
	return string(short)
}
