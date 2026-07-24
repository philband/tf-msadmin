package genframework

import (
	"strings"
	"testing"
)

func roleGroupFixture() (Config, Resource) {
	cfg := Config{
		Package:        "provider",
		ClientsImport:  "github.com/philband/terraform-provider-exo/internal/clients",
		ClientField:    "EXO",
		BindingsImport: "github.com/philband/go-exoscc/exo",
		BindingsPkg:    "exo",
	}
	r := Resource{
		Noun:        "RoleGroup",
		TFName:      "role_group",
		Description: "A management role group.",
		Attributes: []Attribute{
			{TFName: "name", Field: "Name", APIName: "Name", Type: TypeString, Required: true, Replace: true, Description: "Unique name.", InCreate: true},
			{TFName: "description", Field: "Description", APIName: "Description", Type: TypeString, Computed: true, Description: "Description.", InCreate: true, InUpdate: true},
			{TFName: "display_name", Field: "DisplayName", APIName: "DisplayName", Type: TypeString, Computed: true, Description: "Display name.", InCreate: true, InUpdate: true},
			{TFName: "roles", Field: "Roles", APIName: "Roles", Type: TypeStringSet, Computed: true, Replace: true, Description: "Roles.", InCreate: true},
		},
		Create: Op{Method: "NewRoleGroup", Params: "NewRoleGroupParams"},
		Read:   Op{Method: "GetRoleGroup", Params: "GetRoleGroupParams", IdentityField: "Identity"},
		Update: Op{Method: "SetRoleGroup", Params: "SetRoleGroupParams", IdentityField: "Identity"},
		Delete: Op{Method: "RemoveRoleGroup", Params: "RemoveRoleGroupParams", IdentityField: "Identity"},
	}
	return cfg, r
}

func TestGenerateRoleGroupIsValidGo(t *testing.T) {
	cfg, r := roleGroupFixture()
	files, err := Generate(cfg, []Resource{r})
	if err != nil {
		t.Fatalf("Generate: %v", err) // format.Source failure surfaces the bad source
	}
	if len(files) != 3 { // resource + data source + registration
		t.Fatalf("want 3 files, got %d", len(files))
	}

	var res, ds string
	for _, f := range files {
		switch f.Name {
		case "role_group_resource.go":
			res = string(f.Content)
		case "role_group_data_source.go":
			ds = string(f.Content)
		}
	}
	if res == "" || ds == "" {
		t.Fatalf("missing generated files: resource=%v data_source=%v", res != "", ds != "")
	}
	for _, want := range []string{
		"func NewRoleGroupDataSource() datasource.DataSource",
		"readRoleGroup(ctx, obj, &data)",
		"d.client.EXO.GetRoleGroup(ctx",
	} {
		if !strings.Contains(ds, want) {
			t.Errorf("data source missing %q", want)
		}
	}

	for _, want := range []string{
		"func NewRoleGroupResource() resource.Resource",
		"type roleGroupModel struct",
		"exo.NewRoleGroupParams{",
		"r.client.EXO.NewRoleGroup(ctx, p)",
		"resourcex.LoadUntil(ctx, consistency.Config{}, get, reflected)",
		"reconcile.KeepStr(cfg.Description, read.Description)",
		"stringplanmodifier.RequiresReplace()", // name replace
		"setplanmodifier.RequiresReplace()",    // roles replace
		`resp.TypeName = req.ProviderTypeName + "_role_group"`,
	} {
		if !strings.Contains(res, want) {
			t.Errorf("generated source missing %q", want)
		}
	}
}

// TestPointerParamEmitsPointerAccessors covers the tri-state (Nullable<T>) path
// used by the Teams config surface, whose bindings take *bool / *string so an
// explicit false / "" is distinguishable from unset.
func TestPointerParamEmitsPointerAccessors(t *testing.T) {
	cfg := Config{
		Package: "provider", ClientsImport: "example.com/clients", ClientField: "CS",
		BindingsImport: "github.com/philband/go-teams/cs", BindingsPkg: "cs",
	}
	r := Resource{
		Noun: "TeamsMeetingPolicy", TFName: "teams_meeting_policy", Description: "A meeting policy.",
		Attributes: []Attribute{
			{TFName: "allow_meet_now", Field: "AllowMeetNow", APIName: "AllowMeetNow", Type: TypeBool, Computed: true, PointerParam: true, Description: "Allow meet now.", InCreate: true, InUpdate: true},
			{TFName: "description", Field: "Description", APIName: "Description", Type: TypeString, Computed: true, PointerParam: true, Description: "Description.", InCreate: true, InUpdate: true},
		},
		Create: Op{Method: "NewCsTeamsMeetingPolicy", Params: "NewCsTeamsMeetingPolicyParams", IdentityField: "Identity"},
		Read:   Op{Method: "GetCsTeamsMeetingPolicy", Params: "GetCsTeamsMeetingPolicyParams", IdentityField: "Identity"},
		Update: Op{Method: "SetCsTeamsMeetingPolicy", Params: "SetCsTeamsMeetingPolicyParams", IdentityField: "Identity"},
		Delete: Op{Method: "RemoveCsTeamsMeetingPolicy", Params: "RemoveCsTeamsMeetingPolicyParams", IdentityField: "Identity"},
	}
	files, err := Generate(cfg, []Resource{r})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var res string
	for _, f := range files {
		if f.Name == "teams_meeting_policy_resource.go" {
			res = string(f.Content)
		}
	}
	for _, want := range []string{
		"plan.AllowMeetNow.ValueBoolPointer()",
		"plan.Description.ValueStringPointer()",
	} {
		if !strings.Contains(res, want) {
			t.Errorf("generated source missing pointer accessor %q", want)
		}
	}
	if strings.Contains(res, "plan.AllowMeetNow.ValueBool()") {
		t.Error("PointerParam bool must use ValueBoolPointer(), not ValueBool()")
	}
}
