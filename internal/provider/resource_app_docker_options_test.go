package provider

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDokkuAppDockerOptions(t *testing.T) {
	appName := fmt.Sprintf("test-docker-options-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDokkuAppDockerOptionsConfig(appName, `["--label terraform-test=one"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokku_app_docker_options.test", "deploy.#", "1"),
					resource.TestCheckTypeSetElemAttr("dokku_app_docker_options.test", "deploy.*", "--label terraform-test=one"),
				),
			},
			{
				ResourceName:      "dokku_app_docker_options.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccDokkuAppDockerOptionsConfig(appName, `["--label terraform-test=two", "--add-host example.test:127.0.0.1"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokku_app_docker_options.test", "deploy.#", "2"),
					resource.TestCheckTypeSetElemAttr("dokku_app_docker_options.test", "deploy.*", "--label terraform-test=two"),
					resource.TestCheckTypeSetElemAttr("dokku_app_docker_options.test", "deploy.*", "--add-host example.test:127.0.0.1"),
				),
			},
		},
	})
}

func testAccDokkuAppDockerOptionsConfig(appName, deploy string) string {
	return fmt.Sprintf(`
resource "dokku_app" "test" {
  name = %q
}

resource "dokku_app_docker_options" "test" {
  app    = dokku_app.test.name
  deploy = %s
}
`, appName, deploy)
}

func TestParseAppDockerOptionsReport(t *testing.T) {
	stdout := `{"build":"","build-list":[],"deploy":"--cap-add NET_ADMIN -p 25:25","deploy-list":["--cap-add NET_ADMIN","-p 25:25"],"run":"","run-list":[]}`
	got, err := parseAppDockerOptionsReport("mail", stdout)
	if err != nil {
		t.Fatal(err)
	}
	want := dokkuAppDockerOptions{
		Build:  []string{},
		Deploy: []string{"--cap-add NET_ADMIN", "-p 25:25"},
		Run:    []string{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseAppDockerOptionsReportRejectsInvalidJSON(t *testing.T) {
	if _, err := parseAppDockerOptionsReport("mail", `{`); err == nil {
		t.Fatal("expected invalid JSON to return an error")
	}
}
