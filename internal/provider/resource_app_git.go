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

type dokkuAppGitReport struct {
	DeployBranch string `json:"deploy-branch"`
	SourceImage  string `json:"source-image"`
}

func resourceAppGit() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages stable Git deployment properties while reporting deployment-generated source image metadata.",
		CreateContext: resourceAppGitCreate,
		ReadContext:   resourceAppGitRead,
		UpdateContext: resourceAppGitUpdate,
		DeleteContext: resourceAppGitDelete,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Dokku application whose Git properties are managed.",
			},
			"deploy_branch": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "Branch accepted by Dokku's Git receive deployment workflow.",
			},
			"source_image": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current deployment source image. This is pipeline-owned metadata and is not reconciled.",
			},
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppGitCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAppGitDeployBranch(d.Get("app").(string), d.Get("deploy_branch").(string), m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppGitRead(ctx, d, m)
}

func resourceAppGitRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}
	report, exists, err := readAppGitReport(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	d.SetId(app)
	d.Set("app", app)
	d.Set("deploy_branch", report.DeployBranch)
	d.Set("source_image", report.SourceImage)
	return nil
}

func resourceAppGitUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange("deploy_branch") {
		if err := setAppGitDeployBranch(d.Get("app").(string), d.Get("deploy_branch").(string), m.(*goph.Client)); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceAppGitRead(ctx, d, m)
}

func resourceAppGitDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Get("app").(string)
	if res := run(m.(*goph.Client), fmt.Sprintf("git:set %s deploy-branch", shellescape.Quote(app))); res.err != nil {
		return diag.FromErr(res.err)
	}
	d.SetId("")
	return nil
}

func setAppGitDeployBranch(app, branch string, client *goph.Client) error {
	return run(client, fmt.Sprintf(
		"git:set %s deploy-branch %s",
		shellescape.Quote(app),
		shellescape.Quote(branch),
	)).err
}

func readAppGitReport(app string, client *goph.Client) (dokkuAppGitReport, bool, error) {
	if exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app))); exists.err != nil {
		if exists.status > 0 {
			return dokkuAppGitReport{}, false, nil
		}
		return dokkuAppGitReport{}, false, exists.err
	}
	res := run(client, fmt.Sprintf("git:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return dokkuAppGitReport{}, false, res.err
	}
	var report dokkuAppGitReport
	if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
		return dokkuAppGitReport{}, false, fmt.Errorf("parsing git report for %q: %w", app, err)
	}
	return report, true, nil
}
