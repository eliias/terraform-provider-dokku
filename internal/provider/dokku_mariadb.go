package provider

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/melbahja/goph"
)

type DokkuMariadbService struct {
	DokkuGenericService
}

func NewMariadbService(name string) *DokkuMariadbService {
	return &DokkuMariadbService{
		DokkuGenericService: DokkuGenericService{
			Name:    name,
			CmdName: "mariadb",
		},
	}
}

func NewMariadbServiceFromResourceData(d *schema.ResourceData) *DokkuMariadbService {
	return &DokkuMariadbService{
		DokkuGenericService: DokkuGenericService{
			Name:         d.Get("name").(string),
			Image:        d.Get("image").(string),
			ImageVersion: d.Get("image_version").(string),
			Stopped:      d.Get("stopped").(bool),
			Exposed:      strings.Split(d.Get("expose_on").(string), " "),
			CmdName:      "mariadb",
			MemoryMB:     d.Get("memory_mb").(int),
			ShmSize:      d.Get("shm_size").(string),
		},
	}
}

func dokkuMariadbRead(mariadb *DokkuMariadbService, client *goph.Client) error {
	return dokkuServiceRead(&mariadb.DokkuGenericService, client)
}

func dokkuMariadbCreate(mariadb *DokkuMariadbService, client *goph.Client) error {
	return dokkuServiceCreate(&mariadb.DokkuGenericService, client)
}

func dokkuMariadbUpdate(mariadb *DokkuMariadbService, d *schema.ResourceData, client *goph.Client) error {
	return dokkuServiceUpdate(&mariadb.DokkuGenericService, d, client)
}

func dokkuMariadbDestroy(mariadb *DokkuMariadbService, client *goph.Client) error {
	return dokkuServiceDestroy(mariadb.CmdName, mariadb.Name, client)
}
