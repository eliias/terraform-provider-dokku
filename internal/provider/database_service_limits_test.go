package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDatabaseServiceLimitSchemasAreConsistent(t *testing.T) {
	resources := map[string]*schema.Resource{
		"postgres": resourcePostgresService(),
		"redis":    resourceRedisService(),
		"mysql":    resourceMysqlService(),
		"mariadb":  resourceMariadbService(),
	}

	for name, resource := range resources {
		t.Run(name, func(t *testing.T) {
			for _, field := range []string{"memory_mb", "shm_size"} {
				item := resource.Schema[field]
				if item == nil {
					t.Fatalf("missing %s schema", field)
				}
				if !item.Optional || !item.Computed || !item.ForceNew {
					t.Errorf("%s must be optional, computed, and replace-on-change", field)
				}
			}
		})
	}
}

func TestSetDatabaseServiceLimitsFromOutput(t *testing.T) {
	service := &DokkuGenericService{CmdName: "postgres"}
	output := "-----> Filesystem changes may not persist after container restarts\n" +
		"terraform-memory=536870912\n" +
		"terraform-shm=268435456\n"

	if err := setDatabaseServiceLimitsFromOutput(service, output); err != nil {
		t.Fatalf("parse limits: %v", err)
	}
	if service.MemoryMB != 512 {
		t.Errorf("unexpected memory: %d", service.MemoryMB)
	}
	if service.ShmSize != "256m" {
		t.Errorf("unexpected shm size: %q", service.ShmSize)
	}
	if !service.LimitsRead {
		t.Error("expected limits to be marked as read")
	}
}

func TestSetDatabaseServiceLimitsRequiresBothValues(t *testing.T) {
	service := &DokkuGenericService{CmdName: "mysql"}
	if err := setDatabaseServiceLimitsFromOutput(service, "terraform-memory=max\n"); err == nil {
		t.Fatal("expected missing shared-memory output to fail")
	}
}

func TestSetDatabaseServiceLimitsTreatsMaxAsUnlimited(t *testing.T) {
	service := &DokkuGenericService{CmdName: "redis", MemoryMB: 512}
	if err := setDatabaseServiceLimitsFromOutput(service, "terraform-memory=max\nterraform-shm=67108864\n"); err != nil {
		t.Fatalf("parse limits: %v", err)
	}
	if service.MemoryMB != 0 {
		t.Errorf("expected unlimited memory, got %d", service.MemoryMB)
	}
	if service.ShmSize != "64m" {
		t.Errorf("unexpected shm size: %q", service.ShmSize)
	}
}

func TestDockerSizeEquivalence(t *testing.T) {
	for _, test := range []struct {
		left  string
		right string
		equal bool
	}{
		{left: "256m", right: "256mb", equal: true},
		{left: "256m", right: "268435456", equal: true},
		{left: "1g", right: "1024m", equal: true},
		{left: "64m", right: "128m", equal: false},
	} {
		left, leftOK := parseDockerSize(test.left)
		right, rightOK := parseDockerSize(test.right)
		if !leftOK || !rightOK {
			t.Fatalf("failed to parse %q or %q", test.left, test.right)
		}
		if got := left == right; got != test.equal {
			t.Errorf("equivalence for %q and %q: got %t, want %t", test.left, test.right, got, test.equal)
		}
	}
}
