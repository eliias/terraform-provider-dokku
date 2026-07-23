package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

type dokkuAppNetwork struct {
	App                       string
	AttachPostCreate          []string
	AttachPostDeploy          []string
	InitialNetwork            string
	BindAllInterfaces         string
	TLD                       string
	StaticWebListener         string
	ComputedAttachPostCreate  []string
	ComputedAttachPostDeploy  []string
	ComputedInitialNetwork    string
	ComputedBindAllInterfaces bool
	ComputedTLD               string
	WebListeners              []string
}

func resourceAppNetwork() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages explicit Dokku network properties for an application. Attachment changes affect containers created by a subsequent deploy or rebuild.",
		CreateContext: resourceAppNetworkCreate,
		ReadContext:   resourceAppNetworkRead,
		UpdateContext: resourceAppNetworkUpdate,
		DeleteContext: resourceAppNetworkDelete,
		CustomizeDiff: validateAppNetworkDiff,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku application whose network properties are managed.",
			},
			"attach_post_create": networkSetSchema("Networks attached after container creation and before startup. This applies to build, deploy, and run containers."),
			"attach_post_deploy": networkSetSchema("Networks attached to running deploy containers before the proxy is updated."),
			"initial_network": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network assigned when containers are created. This replaces Docker's default bridge for build, deploy, and run containers.",
			},
			"bind_all_interfaces": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "inherit",
				ValidateFunc: validation.StringInSlice([]string{"inherit", "true", "false"}, false),
				Description:  "Whether docker-local publishes web container ports on random host ports bound to 0.0.0.0. Use inherit to clear the app override.",
			},
			"tld": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Network DNS suffix used when constructing application aliases.",
			},
			"static_web_listener": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Static web listener used by proxy integrations, commonly with the null scheduler.",
			},
			"effective_attach_post_create": computedNetworkSetSchema("Effective post-create networks after applying global inheritance."),
			"effective_attach_post_deploy": computedNetworkSetSchema("Effective post-deploy networks after applying global inheritance."),
			"effective_initial_network": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Effective initial network after applying global inheritance.",
			},
			"effective_bind_all_interfaces": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Effective bind-all-interfaces value after applying global inheritance.",
			},
			"effective_tld": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Effective network DNS suffix after applying global inheritance.",
			},
			"web_listeners": computedNetworkSetSchema("Current web-process listeners detected by Dokku."),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func networkSetSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringNotInSlice([]string{"", "bridge", "host"}, false),
		},
		Description: description,
	}
}

func computedNetworkSetSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: description,
	}
}

func validateAppNetworkDiff(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	postCreate := stringSetLookup(d.Get("attach_post_create").(*schema.Set))
	for network := range stringSetLookup(d.Get("attach_post_deploy").(*schema.Set)) {
		if _, duplicate := postCreate[network]; duplicate {
			return fmt.Errorf("network %q cannot be in both attach_post_create and attach_post_deploy", network)
		}
	}
	return nil
}

func stringSetLookup(values *schema.Set) map[string]struct{} {
	lookup := make(map[string]struct{}, values.Len())
	for _, value := range values.List() {
		lookup[value.(string)] = struct{}{}
	}
	return lookup
}

func resourceAppNetworkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllAppNetworkProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppNetworkRead(ctx, d, m)
}

func resourceAppNetworkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}

	network, exists, err := readAppNetwork(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	d.SetId(app)
	d.Set("app", app)
	d.Set("attach_post_create", network.AttachPostCreate)
	d.Set("attach_post_deploy", network.AttachPostDeploy)
	d.Set("initial_network", network.InitialNetwork)
	if network.BindAllInterfaces == "" {
		d.Set("bind_all_interfaces", "inherit")
	} else {
		d.Set("bind_all_interfaces", network.BindAllInterfaces)
	}
	d.Set("tld", network.TLD)
	d.Set("static_web_listener", network.StaticWebListener)
	d.Set("effective_attach_post_create", network.ComputedAttachPostCreate)
	d.Set("effective_attach_post_deploy", network.ComputedAttachPostDeploy)
	d.Set("effective_initial_network", network.ComputedInitialNetwork)
	d.Set("effective_bind_all_interfaces", network.ComputedBindAllInterfaces)
	d.Set("effective_tld", network.ComputedTLD)
	d.Set("web_listeners", network.WebListeners)
	return nil
}

func resourceAppNetworkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	if d.HasChanges("attach_post_create", "attach_post_deploy") {
		if err := replaceAppNetworkAttachments(d, client); err != nil {
			return diag.FromErr(err)
		}
	}
	for _, property := range appNetworkProperties() {
		if property == "attach_post_create" || property == "attach_post_deploy" {
			continue
		}
		if !d.HasChange(property) {
			continue
		}
		if err := setAppNetworkProperty(d, property, client); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceAppNetworkRead(ctx, d, m)
}

func resourceAppNetworkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app)))
	if exists.err != nil {
		if exists.status > 0 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(exists.err)
	}
	for _, property := range appNetworkProperties() {
		res := run(client, fmt.Sprintf("network:set %s %s", shellescape.Quote(app), shellescape.Quote(strings.ReplaceAll(property, "_", "-"))))
		if res.err != nil {
			return diag.FromErr(res.err)
		}
	}
	d.SetId("")
	return nil
}

func replaceAppNetworkAttachments(d *schema.ResourceData, client *goph.Client) error {
	app := shellescape.Quote(d.Get("app").(string))
	for _, property := range []string{"attach-post-create", "attach-post-deploy"} {
		res := run(client, fmt.Sprintf("network:set %s %s", app, property))
		if res.err != nil {
			return res.err
		}
	}
	for _, property := range []string{"attach_post_create", "attach_post_deploy"} {
		if err := setAppNetworkProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAllAppNetworkProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range appNetworkProperties() {
		if err := setAppNetworkProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppNetworkProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	values := appNetworkPropertyValues(d, property)
	args := []string{shellescape.Quote(app), shellescape.Quote(strings.ReplaceAll(property, "_", "-"))}
	for _, value := range values {
		args = append(args, shellescape.Quote(value))
	}

	res := run(client, fmt.Sprintf("network:set %s", strings.Join(args, " ")))
	return res.err
}

func appNetworkPropertyValues(d *schema.ResourceData, property string) []string {
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

func appNetworkProperties() []string {
	return []string{
		"attach_post_create",
		"attach_post_deploy",
		"initial_network",
		"bind_all_interfaces",
		"tld",
		"static_web_listener",
	}
}

func readAppNetwork(app string, client *goph.Client) (dokkuAppNetwork, bool, error) {
	exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app)))
	if exists.err != nil {
		if exists.status > 0 {
			return dokkuAppNetwork{}, false, nil
		}
		return dokkuAppNetwork{}, false, exists.err
	}

	res := run(client, fmt.Sprintf("network:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return dokkuAppNetwork{}, false, res.err
	}
	network, err := parseAppNetworkReport(app, res.stdout)
	if err != nil {
		return dokkuAppNetwork{}, false, err
	}
	return network, true, nil
}

func parseAppNetworkReport(app, stdout string) (dokkuAppNetwork, error) {
	var report map[string]string
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		return dokkuAppNetwork{}, fmt.Errorf("parsing network report for %q: %w", app, err)
	}

	bindAllInterfaces := reportValue(report, "bind-all-interfaces")
	if bindAllInterfaces != "" {
		if _, err := strconv.ParseBool(bindAllInterfaces); err != nil {
			return dokkuAppNetwork{}, fmt.Errorf("parsing network bind-all-interfaces for %q: %w", app, err)
		}
	}

	return dokkuAppNetwork{
		App:                       app,
		AttachPostCreate:          splitNetworkValues(reportValue(report, "attach-post-create")),
		AttachPostDeploy:          splitNetworkValues(reportValue(report, "attach-post-deploy")),
		InitialNetwork:            reportValue(report, "initial-network"),
		BindAllInterfaces:         bindAllInterfaces,
		TLD:                       reportValue(report, "tld"),
		StaticWebListener:         reportValue(report, "static-web-listener"),
		ComputedAttachPostCreate:  splitNetworkValues(reportValue(report, "computed-attach-post-create")),
		ComputedAttachPostDeploy:  splitNetworkValues(reportValue(report, "computed-attach-post-deploy")),
		ComputedInitialNetwork:    reportValue(report, "computed-initial-network"),
		ComputedBindAllInterfaces: parseReportBool(reportValue(report, "computed-bind-all-interfaces")),
		ComputedTLD:               reportValue(report, "computed-tld"),
		WebListeners:              splitNetworkValues(reportValue(report, "web-listeners")),
	}, nil
}

func splitNetworkValues(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func parseReportBool(value string) bool {
	parsed, _ := strconv.ParseBool(value)
	return parsed
}

func reportValue(report map[string]string, key string) string {
	if value, ok := report[key]; ok {
		return value
	}
	return report["network-"+key]
}
