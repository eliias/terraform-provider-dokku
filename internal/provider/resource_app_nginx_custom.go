package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

var managedAppNginxCustomProperties = []string{
	"nginx_conf_sigil_path",
	"disable_custom_config",
}

func resourceAppNginxCustom() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages custom nginx template selection state without managing template file contents or rebuilding generated proxy configuration.",
		CreateContext: resourceAppNginxCustomCreate,
		ReadContext:   resourceAppNginxCustomRead,
		UpdateContext: resourceAppNginxCustomUpdate,
		DeleteContext: resourceAppNginxCustomDelete,
		Schema: map[string]*schema.Schema{
			"app": appSettingsNameSchema("Dokku application whose custom nginx configuration state is managed."),
			"nginx_conf_sigil_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explicit custom nginx sigil template path relative to the app repository. Empty inherits the global and built-in default.",
			},
			"disable_custom_config": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"true", "false"}, false),
				Description:  "Explicit custom-config disable setting. Empty inherits the global and built-in default.",
			},
			"effective_nginx_conf_sigil_path": computedStringSchema("Effective custom nginx sigil template path."),
			"effective_disable_custom_config": computedBoolSchema("Whether custom nginx configuration is effectively disabled."),
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppNginxCustomCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllAppNginxCustomProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppNginxCustomRead(ctx, d, m)
}

func resourceAppNginxCustomRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := appSettingsResourceApp(d)
	report, exists, err := readAppNginxProperties(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	d.SetId(app)
	d.Set("app", app)
	d.Set("nginx_conf_sigil_path", report["nginx-conf-sigil-path"])
	d.Set("disable_custom_config", report["disable-custom-config"])
	d.Set("effective_nginx_conf_sigil_path", report["computed-nginx-conf-sigil-path"])
	d.Set("effective_disable_custom_config", report["computed-disable-custom-config"] == "true")
	return nil
}

func resourceAppNginxCustomUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range managedAppNginxCustomProperties {
		if d.HasChange(property) {
			if err := setAppNginxCustomProperty(d, property, client); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return resourceAppNginxCustomRead(ctx, d, m)
}

func resourceAppNginxCustomDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	for _, property := range managedAppNginxCustomProperties {
		if err := run(client, fmt.Sprintf("nginx:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))).err; err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId("")
	return nil
}

func setAllAppNginxCustomProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range managedAppNginxCustomProperties {
		if err := setAppNginxCustomProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppNginxCustomProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	command := fmt.Sprintf("nginx:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))
	if value := d.Get(property).(string); value != "" {
		command += " " + shellescape.Quote(value)
	}
	return run(client, command).err
}
