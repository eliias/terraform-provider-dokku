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

type dokkuAppBuilderReport struct {
	BuildDir    string `json:"build-dir"`
	Selected    string `json:"selected"`
	SkipCleanup string `json:"skip-cleanup"`
}

func resourceAppBuilder() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages explicit builder properties for a Dokku application. Changes affect future builds and do not trigger a deploy.",
		CreateContext: resourceAppBuilderCreate,
		ReadContext:   resourceAppBuilderRead,
		UpdateContext: resourceAppBuilderUpdate,
		DeleteContext: resourceAppBuilderDelete,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku application whose builder properties are managed.",
			},
			"build_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Subdirectory used as the build context.",
			},
			"selected": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explicit builder name, including null or a custom installed builder.",
			},
			"skip_cleanup": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"true", "false"}, false),
				Description:  "Whether Dokku preserves builder artifacts after a build.",
			},
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppBuilderCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllAppBuilderProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppBuilderRead(ctx, d, m)
}

func resourceAppBuilderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}
	report, exists, err := readAppBuilderReport(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	d.SetId(app)
	d.Set("app", app)
	d.Set("build_dir", report.BuildDir)
	d.Set("selected", report.Selected)
	d.Set("skip_cleanup", report.SkipCleanup)
	return nil
}

func resourceAppBuilderUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range []string{"build_dir", "selected", "skip_cleanup"} {
		if d.HasChange(property) {
			if err := setAppBuilderProperty(d, property, client); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return resourceAppBuilderRead(ctx, d, m)
}

func resourceAppBuilderDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	for _, property := range []string{"build-dir", "selected", "skip-cleanup"} {
		if res := run(client, fmt.Sprintf("builder:set %s %s", shellescape.Quote(app), property)); res.err != nil {
			return diag.FromErr(res.err)
		}
	}
	d.SetId("")
	return nil
}

func setAllAppBuilderProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range []string{"build_dir", "selected", "skip_cleanup"} {
		if err := setAppBuilderProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppBuilderProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	value := d.Get(property).(string)
	command := fmt.Sprintf("builder:set %s %s", shellescape.Quote(app), shellescape.Quote(dokkuPropertyName(property)))
	if value != "" {
		command += " " + shellescape.Quote(value)
	}
	return run(client, command).err
}

func readAppBuilderReport(app string, client *goph.Client) (dokkuAppBuilderReport, bool, error) {
	if exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app))); exists.err != nil {
		if exists.status > 0 {
			return dokkuAppBuilderReport{}, false, nil
		}
		return dokkuAppBuilderReport{}, false, exists.err
	}
	res := run(client, fmt.Sprintf("builder:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return dokkuAppBuilderReport{}, false, res.err
	}
	var report dokkuAppBuilderReport
	if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
		return dokkuAppBuilderReport{}, false, fmt.Errorf("parsing builder report for %q: %w", app, err)
	}
	return report, true, nil
}
