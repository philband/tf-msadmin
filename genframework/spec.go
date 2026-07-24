// Package genframework generates terraform-plugin-framework resource code (and
// docs) from a normalized, provider-agnostic resource description.
//
// It is deliberately decoupled from any particular API client: the client calls
// a generated resource makes are carried as data (package name, method name,
// params type name), so the same engine serves the cmdlet-backed providers
// (Exchange Online, Purview) and, in future, a REST-backed one (Teams). A
// provider supplies a small frontend that turns its own catalog/metadata into
// []Resource and calls Generate.
//
// Generated resources lean on the shared runtime — reconcile, resourcex and
// go-msadmin/consistency — plus a small set of provider-local helpers that form
// the generation contract (see Config).
package genframework

// AttrType is the Terraform attribute type an attribute maps to.
type AttrType int

const (
	TypeString    AttrType = iota // types.String  <- string / *.guid
	TypeBool                      // types.Bool    <- switch / bool
	TypeStringSet                 // types.Set of String <- []string
)

// Attribute is one resource attribute derived from a cmdlet parameter (and its
// read-back field).
type Attribute struct {
	TFName      string   // snake_case Terraform attribute name
	Field       string   // Go field name on the model and the params struct (PascalCase)
	APIName     string   // read-back object field name (PascalCase)
	Type        AttrType // Terraform/Go type
	Required    bool     // Required (mandatory create key)
	Computed    bool     // Computed (server-populated / normalised)
	Sensitive   bool     // Sensitive value
	Replace     bool     // RequiresReplace (create-time-only)
	Description string   // schema description
	InCreate    bool     // pass to the create params
	InUpdate    bool     // pass to the update params
	// Object is true when the underlying cmdlet parameter is a PowerShell
	// System.Object (an `any` field in the generated bindings). Such fields are
	// only assigned when non-empty, so an empty string is not marshalled as a
	// value. Object attributes are always TypeString.
	Object bool
	// PointerParam is true when the binding's params field is a pointer
	// (*bool / *string) rather than a value — the case for tri-state
	// (Nullable<T>) APIs like the Teams config surface, where an explicit false /
	// "" must be distinguishable from unset. Create/update then pass
	// ValueBoolPointer() / ValueStringPointer() (nil when the plan value is
	// null/unknown, so it is simply omitted).
	PointerParam bool
}

// Op is a client operation binding: the generated code calls
// r.client.<Config.ClientField>.<Method>(ctx, <Config.BindingsPkg>.<Params>{...}).
type Op struct {
	Method string // e.g. "NewRoleGroup"
	Params string // e.g. "NewRoleGroupParams"
	// IdentityField is the params field the op targets the object with (e.g.
	// "Identity", "PlaceMailboxId", "ScheduleId"). Empty means the op takes no
	// key — it applies org-wide (e.g. Set-AvailabilityConfig). Not used by
	// Create (New rarely takes a key).
	IdentityField string
}

// MemberCollection describes a members sub-collection managed by companion
// cmdlets (Get-<Noun>Member to read, Update-<Noun>Member -Members to replace the
// whole set — e.g. RoleGroupMember, DistributionGroupMember). It is exposed as a
// single "members" set attribute whose values are member identities.
type MemberCollection struct {
	TFName        string // Terraform attribute name, e.g. "members"
	Field         string // model field name, e.g. "Members"
	Description   string
	ReadMethod    string   // e.g. "GetRoleGroupMember"
	ReadParams    string   // e.g. "GetRoleGroupMemberParams"
	UpdateMethod  string   // e.g. "UpdateRoleGroupMember"
	UpdateParams  string   // e.g. "UpdateRoleGroupMemberParams"
	IdentityField string   // key param on the member cmdlets, e.g. "Identity"
	MembersField  string   // set param on Update-<Noun>Member, e.g. "Members"
	ReadKeys      []string // read-back object fields for a member's identity, e.g. ["PrimarySmtpAddress","Name"]
}

// Resource is a normalized description of one resource to generate.
type Resource struct {
	Noun        string // API noun, e.g. "RoleGroup"
	TFName      string // resource suffix, e.g. "role_group" -> <provider>_role_group
	Description string
	// IdentityReadField is an extra read-back object field to source the identity
	// value from when the key is not the generic "Identity" (e.g.
	// "PlaceMailboxId"). May be empty.
	IdentityReadField string
	Attributes        []Attribute
	Create            Op
	Read              Op
	Update            Op
	Delete            Op
	// Members, when non-nil, adds a members sub-collection managed by companion
	// cmdlets.
	Members *MemberCollection
}

// Config carries the provider-level constants shared by all generated files and
// the generation contract the provider must satisfy.
type Config struct {
	// Package is the Go package name for generated files (e.g. "provider").
	Package string
	// ClientsImport is the import path of the provider's clients package, which
	// must expose a *clients.Client with a field named ClientField.
	ClientsImport string
	// ClientField is the field on *clients.Client that holds the bindings
	// service (e.g. "EXO").
	ClientField string
	// BindingsImport is the import path of the generated bindings (e.g.
	// "github.com/philband/go-exoscc/exo").
	BindingsImport string
	// BindingsPkg is that package's name (e.g. "exo").
	BindingsPkg string

	// The provider must supply these helpers (generation contract), typically in
	// a hand-written helpers.go:
	//   firstObject(v []map[string]any) map[string]any
	//   getString(map[string]any, string) string
	//   getStringSlice(map[string]any, string) []string
	//   getBool(map[string]any, string) bool
	//   firstNonEmptyStr(...string) string
	//   isNotFound(error) bool
	//   toStringSlice(context.Context, types.Set, *diag.Diagnostics) []string
	//   stringSetValue(context.Context, []string) types.Set
	// and a *clients.Client passed via ResourceData.
}
