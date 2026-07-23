package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDatabaseServiceNetworkSchemasAreConsistent(t *testing.T) {
	resources := map[string]*schema.Resource{
		"postgres":   resourcePostgresService(),
		"redis":      resourceRedisService(),
		"mysql":      resourceMysqlService(),
		"mariadb":    resourceMariadbService(),
		"clickhouse": resourceClickhouseService(),
	}

	for name, resource := range resources {
		t.Run(name, func(t *testing.T) {
			for _, field := range []string{"initial_network", "post_create_networks", "post_start_networks"} {
				item := resource.Schema[field]
				if item == nil {
					t.Fatalf("missing %s schema", field)
				}
				if !item.Optional || item.ForceNew {
					t.Errorf("%s must be optional and mutable", field)
				}
			}
		})
	}
}

func TestCreateServiceFlagStrIncludesNetworks(t *testing.T) {
	service := &DokkuGenericService{
		InitialNetwork:     "private net",
		PostCreateNetworks: []string{"metrics", "backend"},
		PostStartNetworks:  []string{"public"},
	}
	flags := createServiceFlagStr(service)

	for _, expected := range []string{
		"--initial-network 'private net'",
		"--post-create-network backend,metrics",
		"--post-start-network public",
	} {
		if !strings.Contains(flags, expected) {
			t.Errorf("expected flags %q to contain %q", flags, expected)
		}
	}
}

func TestParseServiceNetworks(t *testing.T) {
	if got := parseServiceNetworks("-"); got != nil {
		t.Fatalf("expected dash to represent no networks, got %#v", got)
	}
	got := parseServiceNetworks("backend,metrics")
	if len(got) != 2 || got[0] != "backend" || got[1] != "metrics" {
		t.Fatalf("unexpected parsed networks: %#v", got)
	}
}
