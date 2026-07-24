package provider

import "testing"

func TestAppNginxCustomSchema(t *testing.T) {
	resource := resourceAppNginxCustom()

	if !resource.Schema["app"].ForceNew {
		t.Fatal("app must be ForceNew")
	}
	if !resource.Schema["effective_nginx_conf_sigil_path"].Computed {
		t.Fatal("effective nginx template path must be computed")
	}
	if !resource.Schema["effective_disable_custom_config"].Computed {
		t.Fatal("effective disable-custom-config must be computed")
	}
}
