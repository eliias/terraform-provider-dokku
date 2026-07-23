package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/melbahja/goph"
)

func TestAccDokkuStorage(t *testing.T) {
	suffix := acctest.RandString(10)
	appName := fmt.Sprintf("test-storage-app-%s", suffix)
	entryName := fmt.Sprintf("test-storage-%s", suffix)

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDokkuStorageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDokkuStorageConfig(appName, entryName, true, "noexec,nosuid"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDokkuStorageEntry("dokku_storage_entry.test"),
					testAccCheckDokkuStorageMount("dokku_storage_mount.test", true, "noexec,nosuid"),
				),
			},
			{
				ResourceName:      "dokku_storage_entry.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "dokku_storage_mount.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccDokkuStorageConfig(appName, entryName, false, "Z"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDokkuStorageEntry("dokku_storage_entry.test"),
					testAccCheckDokkuStorageMount("dokku_storage_mount.test", false, "Z"),
				),
			},
		},
	})
}

func testAccDokkuStorageConfig(appName, entryName string, readonly bool, volumeOptions string) string {
	return fmt.Sprintf(`
resource "dokku_app" "test" {
  name = %q
}

resource "dokku_storage_entry" "test" {
  name = %q
}

resource "dokku_storage_mount" "test" {
  app            = dokku_app.test.name
  storage_entry  = dokku_storage_entry.test.name
  container_path = "/app/data"
  readonly       = %t
  volume_options = %q
}
`, appName, entryName, readonly, volumeOptions)
}

func testAccCheckDokkuStorageEntry(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		entry, exists, err := readStorageEntry(rs.Primary.ID, testAccProvider.Meta().(*goph.Client))
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("storage entry %q does not exist", rs.Primary.ID)
		}
		if entry.Scheduler != "docker-local" {
			return fmt.Errorf("unexpected scheduler %q", entry.Scheduler)
		}
		return nil
	}
}

func testAccCheckDokkuStorageMount(resourceName string, readonly bool, volumeOptions string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		app := rs.Primary.Attributes["app"]
		entry := rs.Primary.Attributes["storage_entry"]
		containerPath := rs.Primary.Attributes["container_path"]
		mounts, err := readStorageMounts(app, testAccProvider.Meta().(*goph.Client))
		if err != nil {
			return err
		}

		for _, mount := range mounts {
			if mount.StorageEntry == entry && mount.ContainerPath == containerPath {
				if mount.Readonly != readonly {
					return fmt.Errorf("readonly was %t, expected %t", mount.Readonly, readonly)
				}
				if mount.VolumeOptions != volumeOptions {
					return fmt.Errorf("volume options were %q, expected %q", mount.VolumeOptions, volumeOptions)
				}
				return nil
			}
		}
		return fmt.Errorf("storage mount %q was not found", rs.Primary.ID)
	}
}

func testAccCheckDokkuStorageDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*goph.Client)
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "dokku_storage_entry":
			_, exists, err := readStorageEntry(rs.Primary.ID, client)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("storage entry %q still exists", rs.Primary.ID)
			}
		case "dokku_storage_mount":
			mounts, err := readStorageMounts(rs.Primary.Attributes["app"], client)
			if err != nil {
				continue
			}
			for _, mount := range mounts {
				if storageMountID(mount) == rs.Primary.ID {
					return fmt.Errorf("storage mount %q still exists", rs.Primary.ID)
				}
			}
		}
	}
	return nil
}
