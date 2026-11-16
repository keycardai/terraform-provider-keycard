package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestApplicationWorkloadIdentityResource_UpgradeStateV0(t *testing.T) {
	ctx := context.Background()
	r := &ApplicationWorkloadIdentityResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %s", schemaResp.Diagnostics)
	}

	upgrader := r.UpgradeState(ctx)[0]
	priorType := upgrader.PriorSchema.Type().TerraformType(ctx)
	prior := tfsdk.State{
		Schema: *upgrader.PriorSchema,
		Raw: tftypes.NewValue(priorType, map[string]tftypes.Value{
			"id":             tftypes.NewValue(tftypes.String, "acr_123"),
			"zone_id":        tftypes.NewValue(tftypes.String, "zone_123"),
			"application_id": tftypes.NewValue(tftypes.String, "app_123"),
			"provider_id":    tftypes.NewValue(tftypes.String, "prov_123"),
			"subject":        tftypes.NewValue(tftypes.String, "system:serviceaccount:ns:sa"),
		}),
	}

	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
	upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{State: &prior}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade: %s", resp.Diagnostics)
	}

	var got ApplicationWorkloadIdentityModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get: %s", diags)
	}
	want := ApplicationWorkloadIdentityModel{
		ID:                      types.StringValue("acr_123"),
		ZoneID:                  types.StringValue("zone_123"),
		ApplicationID:           types.StringValue("app_123"),
		ProviderID:              types.StringValue("prov_123"),
		Subject:                 types.StringValue("system:serviceaccount:ns:sa"),
		AllowedAudience:         types.StringNull(),
		AuthenticationCondition: types.StringNull(),
		Version:                 types.Int64Value(1),
		ClientID:                types.StringNull(),
	}
	if got != want {
		t.Errorf("upgraded state = %+v, want %+v", got, want)
	}
}

func TestApplicationWorkloadIdentityResource_ModifyPlanCreateSetsVersion2(t *testing.T) {
	ctx := context.Background()
	r := &ApplicationWorkloadIdentityResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %s", schemaResp.Diagnostics)
	}
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	vals := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(attrType, nil)
	}
	vals["zone_id"] = tftypes.NewValue(tftypes.String, "zone_123")
	vals["application_id"] = tftypes.NewValue(tftypes.String, "app_123")
	vals["provider_id"] = tftypes.NewValue(tftypes.String, "prov_123")
	vals["subject"] = tftypes.NewValue(tftypes.String, "system:serviceaccount:ns:sa")
	for _, name := range []string{"id", "version", "client_id"} {
		vals[name] = tftypes.NewValue(objType.AttributeTypes[name], tftypes.UnknownValue)
	}

	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
	resp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("modify plan: %s", resp.Diagnostics)
	}

	var got types.Int64
	if diags := resp.Plan.GetAttribute(ctx, path.Root("version"), &got); diags.HasError() {
		t.Fatalf("get version: %s", diags)
	}
	if !got.Equal(types.Int64Value(2)) {
		t.Errorf("planned version = %s, want 2", got)
	}
}
