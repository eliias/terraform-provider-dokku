package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/melbahja/goph"
)

func resourceMariadbServiceLink() *schema.Resource {
	return &schema.Resource{
		Description:   "Links a Dokku MariaDB service to an application, creating a connection between them and injecting the MariaDB connection details into the application's environment variables.",
		CreateContext: resourceMariadbServiceLinkCreate,
		ReadContext:   resourceMariadbServiceLinkRead,
		DeleteContext: resourceMariadbServiceLinkDelete,
		Schema: map[string]*schema.Schema{
			"service": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the MariaDB service to link to the application.",
			},
			"app": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the Dokku application that will be linked to the MariaDB service.",
			},
			"alias": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Alternative environment variable name to use in exposing credentials to the app.",
			},
			"query_string": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Additional connection parameters to append to the DATABASE_URL environment variable as a query string.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: importServiceLinkState,
		},
	}
}

const mariadbServiceCmd = "mariadb"

func resourceMariadbServiceLinkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := serviceLinkCreate(d, mariadbServiceCmd, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceMariadbServiceLinkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := serviceLinkRead(d, mariadbServiceCmd, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceMariadbServiceLinkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := serviceLinkDelete(d, mariadbServiceCmd, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
