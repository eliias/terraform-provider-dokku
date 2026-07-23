package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/melbahja/goph"
)

func resourceMariadbService() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a MariaDB service in Dokku. Requires the MariaDB Dokku plugin to be installed.",
		CreateContext: resourceMariadbCreate,
		ReadContext:   resourceMariadbRead,
		UpdateContext: resourceMariadbUpdate,
		DeleteContext: resourceMariadbDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the MariaDB service.",
			},
			"image": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The Docker image to use for the MariaDB service. If not specified, Dokku will use its default MariaDB image.",
			},
			"image_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The version of MariaDB to use. If not specified, Dokku will use its default version.",
			},
			"stopped": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether the MariaDB service is stopped. When true, the database service will not be running but data will be preserved.",
			},
			"expose_on": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network address and port to expose the service on. Format is 'host:port' (e.g. '0.0.0.0:8085'). If not specified, the service remains unexposed.",
			},
			"memory_mb": databaseServiceMemorySchema("MariaDB"),
			"shm_size":  databaseServiceShmSizeSchema("MariaDB"),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceMariadbCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mariadb := NewMariadbServiceFromResourceData(d)
	if err := dokkuMariadbCreate(mariadb, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}

	mariadb.setOnResourceData(d)
	return nil
}

func resourceMariadbRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	serviceName := d.Id()
	if serviceName == "" {
		serviceName = d.Get("name").(string)
	}

	mariadb := NewMariadbService(serviceName)
	if err := dokkuMariadbRead(mariadb, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}

	mariadb.setOnResourceData(d)
	return nil
}

func resourceMariadbUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mariadb := NewMariadbServiceFromResourceData(d)
	if err := dokkuMariadbUpdate(mariadb, d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}

	mariadb.setOnResourceData(d)
	return nil
}

func resourceMariadbDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := dokkuMariadbDestroy(NewMariadbService(d.Id()), m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
