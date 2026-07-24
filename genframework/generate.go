package genframework

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
)

// File is one generated output file.
type File struct {
	Name    string // suggested file name, e.g. "role_group_resource.go"
	Content []byte // gofmt-formatted source
}

// Generate produces one resource file per resource plus a registration file
// (zz_generated_resources.go) that lists their constructors. All output is
// gofmt-formatted; a formatting error surfaces the offending source for
// debugging.
func Generate(cfg Config, resources []Resource) ([]File, error) {
	var files []File
	sorted := append([]Resource(nil), resources...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Noun < sorted[j].Noun })

	for _, r := range sorted {
		src, err := genResource(cfg, r)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", r.Noun, err)
		}
		files = append(files, File{Name: r.TFName + "_resource.go", Content: src})
	}

	reg, err := genRegistration(cfg, sorted)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Name: "zz_generated_resources.go", Content: reg})
	return files, nil
}

func gofmt(src string) ([]byte, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("format: %w\n----\n%s", err, src)
	}
	return out, nil
}

// ---- naming helpers ----

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func (r Resource) recv() string  { return lowerFirst(r.Noun) + "Resource" }
func (r Resource) model() string { return lowerFirst(r.Noun) + "Model" }
func (r Resource) ctor() string  { return "New" + r.Noun + "Resource" }
func (r Resource) hasSet() bool  { return r.hasType(TypeStringSet) }

// hasBoolMod reports whether any Bool attribute emits a plan modifier (only
// RequiresReplace does), which requires the boolplanmodifier import.
func (r Resource) hasBoolMod() bool {
	for _, a := range r.Attributes {
		if a.Type == TypeBool && a.planModifiers() != "" {
			return true
		}
	}
	return false
}
func (r Resource) hasName() bool { return r.field("Name") != nil }
func (r Resource) cmdlet(v string) string {
	return v + "-" + r.Noun
}

func (r Resource) hasType(t AttrType) bool {
	for _, a := range r.Attributes {
		if a.Type == t {
			return true
		}
	}
	return false
}

func (r Resource) field(name string) *Attribute {
	for i := range r.Attributes {
		if r.Attributes[i].Field == name {
			return &r.Attributes[i]
		}
	}
	return nil
}

// ---- per-attribute rendering ----

func (a Attribute) tfType() string {
	switch a.Type {
	case TypeBool:
		return "Bool"
	case TypeStringSet:
		return "Set"
	default:
		return "String"
	}
}

func (a Attribute) keepFn() string {
	switch a.Type {
	case TypeBool:
		return "KeepBool"
	case TypeStringSet:
		return "KeepSet"
	default:
		return "KeepStr"
	}
}

// modelField renders the struct field for the model.
func (a Attribute) modelField() string {
	return fmt.Sprintf("%s types.%s `tfsdk:%q`", a.Field, a.tfType(), a.TFName)
}

// schemaAttr renders the schema.Attribute literal.
func (a Attribute) schemaAttr() string {
	var b strings.Builder
	mods := a.planModifiers()
	switch a.Type {
	case TypeBool:
		b.WriteString("schema.BoolAttribute{")
		b.WriteString(a.modeFields())
		if a.Sensitive {
			b.WriteString("Sensitive: true, ")
		}
		fmt.Fprintf(&b, "Description: %q, ", a.Description)
		if mods != "" {
			fmt.Fprintf(&b, "PlanModifiers: []planmodifier.Bool{%s}, ", mods)
		}
		b.WriteString("}")
	case TypeStringSet:
		b.WriteString("schema.SetAttribute{ElementType: types.StringType, ")
		b.WriteString(a.modeFields())
		fmt.Fprintf(&b, "Description: %q, ", a.Description)
		if mods != "" {
			fmt.Fprintf(&b, "PlanModifiers: []planmodifier.Set{%s}, ", mods)
		}
		b.WriteString("}")
	default:
		b.WriteString("schema.StringAttribute{")
		b.WriteString(a.modeFields())
		if a.Sensitive {
			b.WriteString("Sensitive: true, ")
		}
		fmt.Fprintf(&b, "Description: %q, ", a.Description)
		if mods != "" {
			fmt.Fprintf(&b, "PlanModifiers: []planmodifier.String{%s}, ", mods)
		}
		b.WriteString("}")
	}
	return b.String()
}

func (a Attribute) modeFields() string {
	if a.Required {
		return "Required: true, "
	}
	return "Optional: true, Computed: true, "
}

func (a Attribute) planModifiers() string {
	var mods []string
	kind := a.tfType() // String/Bool/Set
	pkg := strings.ToLower(kind) + "planmodifier"
	if a.Replace {
		mods = append(mods, pkg+".RequiresReplace()")
	}
	if a.Computed && a.Type == TypeStringSet {
		mods = append(mods, pkg+".UseStateForUnknown()")
	}
	return strings.Join(mods, ", ")
}

// createValue renders the params assignment value for create/update.
func (a Attribute) planValue() string {
	switch a.Type {
	case TypeBool:
		return "plan." + a.Field + ".ValueBool()"
	case TypeStringSet:
		return "toStringSlice(ctx, plan." + a.Field + ", &resp.Diagnostics)"
	default:
		return "plan." + a.Field + ".ValueString()"
	}
}

// readAssign renders the readInto assignment.
func (a Attribute) readAssign() string {
	switch a.Type {
	case TypeBool:
		return fmt.Sprintf("m.%s = types.BoolValue(getBool(obj, %q))", a.Field, a.APIName)
	case TypeStringSet:
		return fmt.Sprintf("m.%s = stringSetValue(ctx, getStringSlice(obj, %q))", a.Field, a.APIName)
	default:
		return fmt.Sprintf("m.%s = types.StringValue(getString(obj, %q))", a.Field, a.APIName)
	}
}
