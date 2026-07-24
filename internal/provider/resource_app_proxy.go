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

type dokkuAppProxyReport struct {
	Disabled string `json:"disabled"`
	Type     string `json:"type"`
}

func resourceAppProxy() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages explicit proxy selection and enabled state for a Dokku application.",
		CreateContext: resourceAppProxyCreate,
		ReadContext:   resourceAppProxyRead,
		UpdateContext: resourceAppProxyUpdate,
		DeleteContext: resourceAppProxyDelete,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku application whose proxy state is managed.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether proxy integration is enabled for the application.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explicit proxy integration name. An empty value inherits the global selection.",
			},
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppProxyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAppProxy(d, m.(*goph.Client), true); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppProxyRead(ctx, d, m)
}

func resourceAppProxyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}
	report, exists, err := readAppProxyReport(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	d.SetId(app)
	d.Set("app", app)
	d.Set("enabled", report.Disabled != "true")
	d.Set("type", report.Type)
	return nil
}

func resourceAppProxyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAppProxy(d, m.(*goph.Client), false); err != nil {
		return diag.FromErr(err)
	}
	return resourceAppProxyRead(ctx, d, m)
}

func resourceAppProxyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Get("app").(string)
	client := m.(*goph.Client)
	if res := run(client, fmt.Sprintf("proxy:set %s", shellescape.Quote(app))); res.err != nil {
		return diag.FromErr(res.err)
	}
	if res := run(client, fmt.Sprintf("proxy:enable %s", shellescape.Quote(app))); res.err != nil {
		return diag.FromErr(res.err)
	}
	d.SetId("")
	return nil
}

func setAppProxy(d *schema.ResourceData, client *goph.Client, all bool) error {
	app := d.Get("app").(string)
	if all || d.HasChange("type") {
		command := fmt.Sprintf("proxy:set %s", shellescape.Quote(app))
		if proxyType := d.Get("type").(string); proxyType != "" {
			command += " " + shellescape.Quote(proxyType)
		}
		if res := run(client, command); res.err != nil {
			return res.err
		}
	}
	if all || d.HasChange("enabled") {
		action := "disable"
		if d.Get("enabled").(bool) {
			action = "enable"
		}
		if res := run(client, fmt.Sprintf("proxy:%s %s", action, shellescape.Quote(app))); res.err != nil {
			return res.err
		}
	}
	return nil
}

func readAppProxyReport(app string, client *goph.Client) (dokkuAppProxyReport, bool, error) {
	if exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app))); exists.err != nil {
		if exists.status > 0 {
			return dokkuAppProxyReport{}, false, nil
		}
		return dokkuAppProxyReport{}, false, exists.err
	}
	res := run(client, fmt.Sprintf("proxy:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return dokkuAppProxyReport{}, false, res.err
	}
	var report dokkuAppProxyReport
	if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
		return dokkuAppProxyReport{}, false, fmt.Errorf("parsing proxy report for %q: %w", app, err)
	}
	return report, true, nil
}
