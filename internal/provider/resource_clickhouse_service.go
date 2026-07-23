package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

// Had issues with other images and cloning (not implemented at time of writing)
// with clickhouse. PR's welcome to implement this behaviour.
//
// This is therefore a less complete resource than e.g postgres, mysql

func resourceClickhouseService() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a ClickHouse service in Dokku. Requires the ClickHouse Dokku plugin to be installed.",
		CreateContext: resourceChCreate,
		ReadContext:   resourceChRead,
		UpdateContext: resourceChUpdate,
		DeleteContext: resourceChDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the ClickHouse service.",
			},
			"stopped": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether the ClickHouse service is stopped. When true, the database service will not be running but data will be preserved.",
			},
			"initial_network":      databaseServiceInitialNetworkSchema("ClickHouse"),
			"post_create_networks": databaseServicePostNetworkSchema("ClickHouse", "after creation"),
			"post_start_networks":  databaseServicePostNetworkSchema("ClickHouse", "after startup"),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceChCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sshClient := m.(*goph.Client)

	var diags diag.Diagnostics

	service := clickhouseServiceFromResourceData(d)
	res := run(sshClient, fmt.Sprintf("clickhouse:create %s %s", shellescape.Quote(service.Name), createServiceFlagStr(service)))

	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId(d.Get("name").(string))

	if d.Get("stopped").(bool) {
		res = run(sshClient, fmt.Sprintf("clickhouse:stop %s", d.Id()))

		if res.err != nil {
			return diag.FromErr(res.err)
		}
	}

	return diags
}

func resourceChRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sshClient := m.(*goph.Client)

	var diags diag.Diagnostics

	serviceInfo, err := getServiceInfo("clickhouse", d.Id(), sshClient)

	if err != nil {
		return diag.FromErr(err)
	}

	if serviceInfo == nil {
		d.SetId("")
		return diags
	}

	if status, ok := serviceInfo["status"]; ok {
		d.Set("stopped", status == "exited" || status == "missing")
	}
	d.Set("initial_network", serviceInfo["initial network"])
	d.Set("post_create_networks", parseServiceNetworks(serviceInfo["post create network"]))
	d.Set("post_start_networks", parseServiceNetworks(serviceInfo["post start network"]))

	return diags
}

func resourceChUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sshClient := m.(*goph.Client)

	var diags diag.Diagnostics

	service := clickhouseServiceFromResourceData(d)
	networkRestartHandled, err := updateDatabaseServiceNetworks(service, d, sshClient)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("stopped") && !networkRestartHandled {
		var res SshOutput

		isStopped := d.Get("stopped").(bool)
		if isStopped {
			res = run(sshClient, fmt.Sprintf("clickhouse:stop %s", d.Id()))
		} else {
			res = run(sshClient, fmt.Sprintf("clickhouse:start %s", d.Id()))
		}

		if res.err != nil {
			return diag.FromErr(res.err)
		}
	}

	return diags
}

func clickhouseServiceFromResourceData(d *schema.ResourceData) *DokkuGenericService {
	return &DokkuGenericService{
		Name:               d.Get("name").(string),
		CmdName:            "clickhouse",
		Stopped:            d.Get("stopped").(bool),
		InitialNetwork:     d.Get("initial_network").(string),
		PostCreateNetworks: sortedStringSet(d, "post_create_networks"),
		PostStartNetworks:  sortedStringSet(d, "post_start_networks"),
	}
}

func sortedStringSet(d *schema.ResourceData, key string) []string {
	values := interfaceSliceToStrSlice(d.Get(key).(*schema.Set).List())
	sort.Strings(values)
	return values
}

func resourceChDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	sshClient := m.(*goph.Client)

	var diags diag.Diagnostics

	res := run(sshClient, fmt.Sprintf("clickhouse:destroy %s -f", d.Id()))

	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId("")

	return diags
}
