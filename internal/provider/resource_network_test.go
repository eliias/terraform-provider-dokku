package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/melbahja/goph"
)

func TestAccDokkuNetwork(t *testing.T) {
	suffix := acctest.RandString(10)
	appName := fmt.Sprintf("test-network-app-%s", suffix)
	networkName := fmt.Sprintf("test-network-%s", suffix)

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDokkuNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDokkuNetworkConfig(appName, networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokku_network.test", "driver", "bridge"),
					resource.TestCheckResourceAttr("dokku_network.test", "dokku_managed", "true"),
					resource.TestCheckResourceAttr("dokku_app_network.test", "attach_post_deploy.#", "1"),
					resource.TestCheckTypeSetElemAttr("dokku_app_network.test", "attach_post_deploy.*", networkName),
				),
			},
			{
				ResourceName:      "dokku_network.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "dokku_app_network.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDokkuNetworkConfig(appName, networkName string) string {
	return fmt.Sprintf(`
resource "dokku_app" "test" {
  name = %q
}

resource "dokku_network" "test" {
  name = %q
}

resource "dokku_app_network" "test" {
  app                = dokku_app.test.name
  attach_post_deploy = [dokku_network.test.name]
}

data "dokku_network" "test" {
  name = dokku_network.test.name
}
`, appName, networkName)
}

func testAccCheckDokkuNetworkDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*goph.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "dokku_network" {
			continue
		}
		_, exists, err := readManagedNetwork(rs.Primary.ID, client)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("network %q still exists", rs.Primary.ID)
		}
	}
	return nil
}
