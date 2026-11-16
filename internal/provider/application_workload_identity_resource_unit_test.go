package provider

import (
	"context"
	"testing"

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
