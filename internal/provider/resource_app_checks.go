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

type dokkuAppChecksReport struct {
	DisabledList string `json:"disabled-list"`
	SkippedList  string `json:"skipped-list"`
}

func resourceAppChecks() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages whether zero-downtime deployment checks are disabled for all processes in a Dokku application.",
		CreateContext: resourceAppChecksCreate,
		ReadContext:   resourceAppChecksRead,
		UpdateContext: resourceAppChecksUpdate,
		DeleteContext: resourceAppChecksDelete,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku application whose deployment-check state is managed.",
			},
			"disabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether zero-downtime checks are disabled for every process type.",
			},
			"skipped_processes": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Raw comma-separated list of process types whose checks are skipped.",
			},
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppChecksCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAppChecksDisabled(d.Get("app").(string), d.Get("disabled").(bool), m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppChecksRead(ctx, d, m)
}

func resourceAppChecksRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}
	report, exists, err := readAppChecksReport(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	d.SetId(app)
	d.Set("app", app)
	d.Set("disabled", report.DisabledList == "_all_")
	d.Set("skipped_processes", report.SkippedList)
	return nil
}

func resourceAppChecksUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("disabled") {
		if err := setAppChecksDisabled(d.Get("app").(string), d.Get("disabled").(bool), m.(*goph.Client)); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceAppChecksRead(ctx, d, m)
}

func resourceAppChecksDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAppChecksDisabled(d.Get("app").(string), false, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func setAppChecksDisabled(app string, disabled bool, client *goph.Client) error {
	action := "enable"
	if disabled {
		action = "disable"
	}
	return run(client, fmt.Sprintf("checks:%s %s", action, shellescape.Quote(app))).err
}

func readAppChecksReport(app string, client *goph.Client) (dokkuAppChecksReport, bool, error) {
	if exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app))); exists.err != nil {
		if exists.status > 0 {
			return dokkuAppChecksReport{}, false, nil
		}
		return dokkuAppChecksReport{}, false, exists.err
	}
	res := run(client, fmt.Sprintf("checks:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return dokkuAppChecksReport{}, false, res.err
	}
	var report dokkuAppChecksReport
	if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
		return dokkuAppChecksReport{}, false, fmt.Errorf("parsing checks report for %q: %w", app, err)
	}
	return report, true, nil
}
