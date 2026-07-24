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
		Noun:          "RoleGroup",
		TFName:        "role_group",
		Description:   "A management role group.",
		IdentityParam: "Identity",
		Attributes: []Attribute{
			{TFName: "name", Field: "Name", APIName: "Name", Type: TypeString, Required: true, Replace: true, Description: "Unique name.", InCreate: true},
			{TFName: "description", Field: "Description", APIName: "Description", Type: TypeString, Computed: true, Description: "Description.", InCreate: true, InUpdate: true},
			{TFName: "display_name", Field: "DisplayName", APIName: "DisplayName", Type: TypeString, Computed: true, Description: "Display name.", InCreate: true, InUpdate: true},
			{TFName: "roles", Field: "Roles", APIName: "Roles", Type: TypeStringSet, Computed: true, Replace: true, Description: "Roles.", InCreate: true},
		},
		Create: Op{Method: "NewRoleGroup", Params: "NewRoleGroupParams"},
		Read:   Op{Method: "GetRoleGroup", Params: "GetRoleGroupParams"},
		Update: Op{Method: "SetRoleGroup", Params: "SetRoleGroupParams"},
		Delete: Op{Method: "RemoveRoleGroup", Params: "RemoveRoleGroupParams"},
	}
	return cfg, r
}

func TestGenerateRoleGroupIsValidGo(t *testing.T) {
	cfg, r := roleGroupFixture()
	files, err := Generate(cfg, []Resource{r})
	if err != nil {
		t.Fatalf("Generate: %v", err) // format.Source failure surfaces the bad source
	}
	if len(files) != 2 { // resource + registration
		t.Fatalf("want 2 files, got %d", len(files))
	}

	var res string
	for _, f := range files {
		if f.Name == "role_group_resource.go" {
			res = string(f.Content)
		}
	}
	if res == "" {
		t.Fatal("role_group_resource.go not generated")
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
