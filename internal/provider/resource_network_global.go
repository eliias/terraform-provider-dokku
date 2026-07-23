package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

type dokkuNetworkGlobal struct {
	AttachPostCreate           []string
	AttachPostDeploy           []string
	InitialNetwork             string
	BindAllInterfaces          string
	TLD                        string
	EffectiveAttachPostCreate  []string
	EffectiveAttachPostDeploy  []string
	EffectiveInitialNetwork    string
	EffectiveBindAllInterfaces bool
	EffectiveTLD               string
}

func resourceNetworkGlobal() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages global Dokku network defaults inherited by applications without explicit app-level properties.",
		CreateContext: resourceNetworkGlobalCreate,
		ReadContext:   resourceNetworkGlobalRead,
		UpdateContext: resourceNetworkGlobalUpdate,
		DeleteContext: resourceNetworkGlobalDelete,
		CustomizeDiff: validateGlobalNetworkDiff,
		Schema: map[string]*schema.Schema{
			"attach_post_create": networkSetSchema("Default networks attached after container creation and before startup."),
			"attach_post_deploy": networkSetSchema("Default networks attached after deployment and before the proxy update."),
			"initial_network": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Default network assigned when containers are created.",
			},
			"bind_all_interfaces": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "inherit",
				ValidateFunc: validation.StringInSlice([]string{"inherit", "true", "false"}, false),
				Description:  "Whether docker-local publishes web container ports on random host ports bound to 0.0.0.0 by default. Use inherit to clear the global property.",
			},
			"tld": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Default network DNS suffix for automatic app/process aliases.",
			},
			"effective_attach_post_create": computedNetworkSetSchema("Effective global post-create networks."),
			"effective_attach_post_deploy": computedNetworkSetSchema("Effective global post-deploy networks."),
			"effective_initial_network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Effective global initial network.",
			},
			"effective_bind_all_interfaces": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Effective global bind-all-interfaces value.",
			},
			"effective_tld": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Effective global network DNS suffix.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func validateGlobalNetworkDiff(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	postCreate := stringSetLookup(d.Get("attach_post_create").(*schema.Set))
	for network := range stringSetLookup(d.Get("attach_post_deploy").(*schema.Set)) {
		if _, duplicate := postCreate[network]; duplicate {
			return fmt.Errorf("network %q cannot be in both attach_post_create and attach_post_deploy", network)
		}
	}
	return nil
}

func resourceNetworkGlobalCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllGlobalNetworkProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("global")
	return resourceNetworkGlobalRead(ctx, d, m)
}

func resourceNetworkGlobalRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	network, err := readGlobalNetwork(m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("global")
	d.Set("attach_post_create", network.AttachPostCreate)
	d.Set("attach_post_deploy", network.AttachPostDeploy)
	d.Set("initial_network", network.InitialNetwork)
	if network.BindAllInterfaces == "" {
		d.Set("bind_all_interfaces", "inherit")
	} else {
		d.Set("bind_all_interfaces", network.BindAllInterfaces)
	}
	d.Set("tld", network.TLD)
	d.Set("effective_attach_post_create", network.EffectiveAttachPostCreate)
	d.Set("effective_attach_post_deploy", network.EffectiveAttachPostDeploy)
	d.Set("effective_initial_network", network.EffectiveInitialNetwork)
	d.Set("effective_bind_all_interfaces", network.EffectiveBindAllInterfaces)
	d.Set("effective_tld", network.EffectiveTLD)
	return nil
}

func resourceNetworkGlobalUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	if d.HasChanges("attach_post_create", "attach_post_deploy") {
		if err := replaceGlobalNetworkAttachments(d, client); err != nil {
			return diag.FromErr(err)
		}
	}
	for _, property := range globalNetworkProperties() {
		if property == "attach_post_create" || property == "attach_post_deploy" {
			continue
		}
		if !d.HasChange(property) {
			continue
		}
		if err := setGlobalNetworkProperty(d, property, client); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceNetworkGlobalRead(ctx, d, m)
}

func replaceGlobalNetworkAttachments(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range []string{"attach-post-create", "attach-post-deploy"} {
		res := run(client, fmt.Sprintf("network:set --global %s", property))
		if res.err != nil {
			return res.err
		}
	}
	for _, property := range []string{"attach_post_create", "attach_post_deploy"} {
		if err := setGlobalNetworkProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func resourceNetworkGlobalDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range globalNetworkProperties() {
		res := run(client, fmt.Sprintf("network:set --global %s", shellescape.Quote(strings.ReplaceAll(property, "_", "-"))))
		if res.err != nil {
			return diag.FromErr(res.err)
		}
	}
	d.SetId("")
	return nil
}

func setAllGlobalNetworkProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range globalNetworkProperties() {
		if err := setGlobalNetworkProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setGlobalNetworkProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	values := globalNetworkPropertyValues(d, property)
	args := []string{"--global", shellescape.Quote(strings.ReplaceAll(property, "_", "-"))}
	for _, value := range values {
		args = append(args, shellescape.Quote(value))
	}
	res := run(client, fmt.Sprintf("network:set %s", strings.Join(args, " ")))
	return res.err
}

func globalNetworkPropertyValues(d *schema.ResourceData, property string) []string {
	switch property {
	case "attach_post_create", "attach_post_deploy":
		values := interfaceSliceToStrSlice(d.Get(property).(*schema.Set).List())
		sort.Strings(values)
		return values
	case "bind_all_interfaces":
		value := d.Get(property).(string)
		if value == "inherit" {
			return nil
		}
		return []string{value}
	default:
		value := d.Get(property).(string)
		if value == "" {
			return nil
		}
		return []string{value}
	}
}

func globalNetworkProperties() []string {
	return []string{"attach_post_create", "attach_post_deploy", "initial_network", "bind_all_interfaces", "tld"}
}

func readGlobalNetwork(client *goph.Client) (dokkuNetworkGlobal, error) {
	res := run(client, "network:report --global --format json")
	if res.err != nil {
		return dokkuNetworkGlobal{}, res.err
	}

	var report map[string]string
	if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
		return dokkuNetworkGlobal{}, fmt.Errorf("parsing global network report: %w", err)
	}
	return dokkuNetworkGlobal{
		AttachPostCreate:           splitNetworkValues(globalReportValue(report, "attach-post-create")),
		AttachPostDeploy:           splitNetworkValues(globalReportValue(report, "attach-post-deploy")),
		InitialNetwork:             globalReportValue(report, "initial-network"),
		BindAllInterfaces:          globalReportValue(report, "bind-all-interfaces"),
		TLD:                        globalReportValue(report, "tld"),
		EffectiveAttachPostCreate:  splitNetworkValues(reportValue(report, "computed-attach-post-create")),
		EffectiveAttachPostDeploy:  splitNetworkValues(reportValue(report, "computed-attach-post-deploy")),
		EffectiveInitialNetwork:    reportValue(report, "computed-initial-network"),
		EffectiveBindAllInterfaces: parseReportBool(reportValue(report, "computed-bind-all-interfaces")),
		EffectiveTLD:               reportValue(report, "computed-tld"),
	}, nil
}

func globalReportValue(report map[string]string, key string) string {
	if value, ok := report["global-"+key]; ok {
		return value
	}
	return report["network-global-"+key]
}
