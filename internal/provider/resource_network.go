package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

type dokkuNetwork struct {
	CreatedAt    string            `json:"CreatedAt"`
	DokkuManaged bool              `json:"DokkuManaged"`
	Driver       string            `json:"Driver"`
	ID           string            `json:"ID"`
	Internal     bool              `json:"Internal"`
	IPv6         bool              `json:"IPv6"`
	Labels       map[string]string `json:"Labels"`
	Name         string            `json:"Name"`
	Scope        string            `json:"Scope"`
}

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages an attachable Docker network created by Dokku. Docker refuses destruction while containers remain attached; use lifecycle.prevent_destroy in production.",
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		DeleteContext: resourceNetworkDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku network name.",
			},
			"driver": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Docker network driver.",
			},
			"scope": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Docker network scope.",
			},
			"internal": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the Docker network is internal.",
			},
			"ipv6": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether IPv6 is enabled for the Docker network.",
			},
			"dokku_managed": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether Dokku created and owns the network.",
			},
			"network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Docker network ID.",
			},
			"labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Docker network labels.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func dataSourceNetwork() *schema.Resource {
	network := resourceNetwork()
	network.Description = "Reads metadata for any existing Docker network visible to Dokku, including built-in and externally managed networks."
	network.CreateContext = nil
	network.UpdateContext = nil
	network.DeleteContext = nil
	network.Importer = nil
	network.ReadContext = dataSourceNetworkRead
	network.Schema["name"].ForceNew = false
	return network
}

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	network, exists, err := readNetworkInfo(name, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		return diag.Errorf("network %q does not exist", name)
	}
	setNetworkOnResourceData(d, network)
	return nil
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	res := run(m.(*goph.Client), fmt.Sprintf("network:create %s", shellescape.Quote(name)))
	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId(name)
	return resourceNetworkRead(ctx, d, m)
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Id()
	if name == "" {
		name = d.Get("name").(string)
	}

	network, exists, err := readManagedNetwork(name, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	if !network.DokkuManaged {
		return diag.Errorf("network %q is not managed by Dokku; use the dokku_network data source instead", name)
	}

	setNetworkOnResourceData(d, network)
	return nil
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	network, exists, err := readManagedNetwork(d.Id(), client)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	if !network.DokkuManaged {
		return diag.Errorf("refusing to destroy network %q because it is not managed by Dokku", d.Id())
	}

	res := run(client, fmt.Sprintf("network:destroy --force %s", shellescape.Quote(d.Id())))
	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId("")
	return nil
}

func setNetworkOnResourceData(d *schema.ResourceData, network dokkuNetwork) {
	d.SetId(network.Name)
	d.Set("name", network.Name)
	d.Set("driver", network.Driver)
	d.Set("scope", network.Scope)
	d.Set("internal", network.Internal)
	d.Set("ipv6", network.IPv6)
	d.Set("dokku_managed", network.DokkuManaged)
	d.Set("network_id", network.ID)
	d.Set("labels", network.Labels)
}

func readNetworkInfo(name string, client *goph.Client) (dokkuNetwork, bool, error) {
	res := run(client, fmt.Sprintf("network:info %s --format json", shellescape.Quote(name)))
	if res.err != nil {
		if res.status > 0 {
			return dokkuNetwork{}, false, nil
		}
		return dokkuNetwork{}, false, res.err
	}

	var network dokkuNetwork
	if err := json.Unmarshal([]byte(res.stdout), &network); err != nil {
		return dokkuNetwork{}, false, fmt.Errorf("parsing network %q: %w", name, err)
	}
	if network.Name == "" {
		return dokkuNetwork{}, false, nil
	}
	return network, true, nil
}

func readManagedNetwork(name string, client *goph.Client) (dokkuNetwork, bool, error) {
	res := run(client, "network:list --format json")
	if res.err != nil {
		return dokkuNetwork{}, false, res.err
	}

	var networks []dokkuNetwork
	if err := json.Unmarshal([]byte(res.stdout), &networks); err != nil {
		return dokkuNetwork{}, false, fmt.Errorf("parsing network list: %w", err)
	}
	for _, network := range networks {
		if network.Name == name {
			return network, true, nil
		}
	}
	return dokkuNetwork{}, false, nil
}
