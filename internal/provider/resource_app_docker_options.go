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

type dokkuAppDockerOptions struct {
	Build  []string `json:"build-list"`
	Deploy []string `json:"deploy-list"`
	Run    []string `json:"run-list"`
}

func resourceAppDockerOptions() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages the authoritative phase-scoped Docker options for a Dokku application. Changes affect subsequently created containers and do not trigger a deploy or rebuild.",
		CreateContext: resourceAppDockerOptionsCreate,
		ReadContext:   resourceAppDockerOptionsRead,
		UpdateContext: resourceAppDockerOptionsUpdate,
		DeleteContext: resourceAppDockerOptionsDelete,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku application whose Docker options are managed.",
			},
			"build":  dockerOptionsPhaseSchema("Docker options applied to build containers."),
			"deploy": dockerOptionsPhaseSchema("Docker options applied to deployed application containers."),
			"run":    dockerOptionsPhaseSchema("Docker options applied to one-off run containers."),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func dockerOptionsPhaseSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validation.StringIsNotEmpty},
		Description: description,
	}
}

func resourceAppDockerOptionsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := replaceAppDockerOptions(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppDockerOptionsRead(ctx, d, m)
}

func resourceAppDockerOptionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}

	options, exists, err := readAppDockerOptions(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	d.SetId(app)
	if err := d.Set("app", app); err != nil {
		return diag.FromErr(err)
	}
	for phase, values := range map[string][]string{
		"build": options.Build, "deploy": options.Deploy, "run": options.Run,
	} {
		if err := d.Set(phase, values); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func resourceAppDockerOptionsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := replaceAppDockerOptions(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	return resourceAppDockerOptionsRead(ctx, d, m)
}

func resourceAppDockerOptionsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
	if err := clearAppDockerOptions(app, client); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func replaceAppDockerOptions(d *schema.ResourceData, client *goph.Client) error {
	app := d.Get("app").(string)
	if err := clearAppDockerOptions(app, client); err != nil {
		return err
	}
	for _, phase := range []string{"build", "deploy", "run"} {
		for _, option := range interfaceSliceToStrSlice(d.Get(phase).(*schema.Set).List()) {
			res := run(client, fmt.Sprintf(
				"docker-options:add %s %s %s",
				shellescape.Quote(app),
				shellescape.Quote(phase),
				shellescape.Quote(option),
			))
			if res.err != nil {
				return res.err
			}
		}
	}
	return nil
}

func clearAppDockerOptions(app string, client *goph.Client) error {
	for _, phase := range []string{"build", "deploy", "run"} {
		res := run(client, fmt.Sprintf(
			"docker-options:clear %s %s",
			shellescape.Quote(app),
			shellescape.Quote(phase),
		))
		if res.err != nil {
			return res.err
		}
	}
	return nil
}

func readAppDockerOptions(app string, client *goph.Client) (dokkuAppDockerOptions, bool, error) {
	exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app)))
	if exists.err != nil {
		if exists.status > 0 {
			return dokkuAppDockerOptions{}, false, nil
		}
		return dokkuAppDockerOptions{}, false, exists.err
	}

	res := run(client, fmt.Sprintf("docker-options:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return dokkuAppDockerOptions{}, false, res.err
	}
	options, err := parseAppDockerOptionsReport(app, res.stdout)
	if err != nil {
		return dokkuAppDockerOptions{}, false, err
	}
	return options, true, nil
}

func parseAppDockerOptionsReport(app, stdout string) (dokkuAppDockerOptions, error) {
	var options dokkuAppDockerOptions
	if err := json.Unmarshal([]byte(stdout), &options); err != nil {
		return dokkuAppDockerOptions{}, fmt.Errorf("parsing docker options report for %q: %w", app, err)
	}
	return options, nil
}
