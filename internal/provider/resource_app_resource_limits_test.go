package provider

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/melbahja/goph"
)

func TestAccDokkuAppResourceLimits(t *testing.T) {
	appName := fmt.Sprintf("test-resource-limits-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccDokkuAppDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "dokku_app" "test" {
  name = "%s"

  resource_limits {
    cpu    = "0.5"
    memory = "256m"
  }

  resource_limits {
    process_type = "web"
    memory       = "128m"
  }
}
`, appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDokkuAppResourceLimit("dokku_app.test", "_default_", DokkuAppResourceLimit{
						CPU:    "0.5",
						Memory: "256m",
					}),
					testAccCheckDokkuAppResourceLimit("dokku_app.test", "web", DokkuAppResourceLimit{
						Memory: "128m",
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "dokku_app" "test" {
  name = "%s"

  resource_limits {
    memory = "512m"
  }
}
`, appName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDokkuAppResourceLimit("dokku_app.test", "_default_", DokkuAppResourceLimit{
						Memory: "512m",
					}),
					testAccCheckDokkuAppResourceLimitAbsent("dokku_app.test", "web"),
				),
			},
		},
	})
}

func testAccCheckDokkuAppResourceLimit(n string, processType string, expected DokkuAppResourceLimit) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		sshClient := testAccProvider.Meta().(*goph.Client)
		app, err := dokkuAppRetrieve(rs.Primary.ID, sshClient)
		if err != nil {
			return fmt.Errorf("retrieving app resource limits: %w", err)
		}

		actual, ok := resourceLimitLookup(app.ResourceLimits)[processType]
		if !ok {
			return fmt.Errorf("resource limits for process type %q not found", processType)
		}
		expected.ProcessType = processType
		if diff := cmp.Diff(expected, actual); diff != "" {
			return fmt.Errorf("unexpected resource limits (-want +got):\n%s", diff)
		}
		return nil
	}
}

func testAccCheckDokkuAppResourceLimitAbsent(n string, processType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		sshClient := testAccProvider.Meta().(*goph.Client)
		app, err := dokkuAppRetrieve(rs.Primary.ID, sshClient)
		if err != nil {
			return fmt.Errorf("retrieving app resource limits: %w", err)
		}

		if _, ok := resourceLimitLookup(app.ResourceLimits)[processType]; ok {
			return fmt.Errorf("resource limits for process type %q still exist", processType)
		}
		return nil
	}
}
