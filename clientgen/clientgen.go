package clientgen

import (
	"bytes"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/formatter"
)

var _ plugin.CodeGenerator = &clientgen{}

// New returns a plugin that generates a graphql go client
func New(dest string) plugin.Plugin {
	return &clientgen{dest}
}

type clientgen struct {
	dest string
}

func (c *clientgen) Name() string {
	return "clientgen"
}

func (c *clientgen) GenerateCode(data *codegen.Data) error {
	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Filename:        c.dest,
		Data:            &Data{data},
		GeneratedHeader: true,
	})
}

// Data is the structure that is given
// to the template that generates the client
type Data struct {
	*codegen.Data
}

// QueryRequest returns a string of the query for
// a mutation or a query field that selects all
// fields of the return type
func (d *Data) QueryRequest(f *codegen.Field, opName ast.Operation) string {
	var op ast.OperationDefinition
	op.Operation = opName
	mainSelection := &ast.Field{
		Name: f.Name,
	}
	op.SelectionSet = []ast.Selection{mainSelection}
	for _, arg := range f.FieldDefinition.Arguments {
		op.VariableDefinitions = append(op.VariableDefinitions, &ast.VariableDefinition{
			Variable:     arg.Name,
			Type:         arg.Type,
			DefaultValue: arg.DefaultValue,
		})
		mainSelection.Arguments = append(mainSelection.Arguments, &ast.Argument{
			Name: arg.Name,
			Value: &ast.Value{
				Raw:  arg.Name,
				Kind: ast.Variable,
			},
		})
	}

	obj := d.Data.Objects.ByName(f.FieldDefinition.Type.Name())
	if obj != nil {
		for _, fieldDef := range obj.Definition.Fields {
			sel := getSelection(d.Data, fieldDef)
			mainSelection.SelectionSet = append(mainSelection.SelectionSet, sel)
		}
	}
	// todo: basic types and arrays?
	var buf bytes.Buffer
	formatter.NewFormatter(&buf).FormatQueryDocument(&ast.QueryDocument{Operations: ast.OperationList{&op}})
	return strings.TrimSpace(buf.String())
}

// GoReturnType returns the return type of a field in Go format
func (d *Data) GoReturnType(f *codegen.Field) string {
	return templates.CurrentImports.LookupType(f.TypeReference.GO)
}

func getSelection(data *codegen.Data, fieldDef *ast.FieldDefinition) ast.Selection {
	f := &ast.Field{Name: fieldDef.Name}
	if obj := data.Objects.ByName(fieldDef.Type.Name()); obj != nil {
		for _, fieldDef := range obj.Definition.Fields {
			f.SelectionSet = append(f.SelectionSet, getSelection(data, fieldDef))
		}
	}
	return f
}
