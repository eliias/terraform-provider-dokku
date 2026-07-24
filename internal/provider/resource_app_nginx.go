package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

var managedAppNginxProperties = []string{
	"client_max_body_size",
	"proxy_buffer_size",
	"proxy_buffers",
	"proxy_busy_buffers_size",
}

func resourceAppNginx() *schema.Resource {
	properties := map[string]*schema.Schema{
		"app": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Dokku application whose nginx properties are managed.",
		},
	}
	for _, property := range managedAppNginxProperties {
		properties[property] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Explicit nginx " + strings.ReplaceAll(property, "_", "-") + " value.",
		}
	}
	return &schema.Resource{
		Description:   "Manages explicit nginx tuning properties for a Dokku application without owning host-level custom configuration files.",
		CreateContext: resourceAppNginxCreate,
		ReadContext:   resourceAppNginxRead,
		UpdateContext: resourceAppNginxUpdate,
		DeleteContext: resourceAppNginxDelete,
		Schema:        properties,
		Importer:      &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppNginxCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllAppNginxProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppNginxRead(ctx, d, m)
}

func resourceAppNginxRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := d.Id()
	if app == "" {
		app = d.Get("app").(string)
	}
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
	for _, property := range managedAppNginxProperties {
		d.Set(property, report[dokkuPropertyName(property)])
	}
	return nil
}

func resourceAppNginxUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range managedAppNginxProperties {
		if d.HasChange(property) {
			if err := setAppNginxProperty(d, property, client); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return resourceAppNginxRead(ctx, d, m)
}

func resourceAppNginxDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	for _, property := range managedAppNginxProperties {
		if res := run(client, fmt.Sprintf("nginx:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))); res.err != nil {
			return diag.FromErr(res.err)
		}
	}
	d.SetId("")
	return nil
}

func setAllAppNginxProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range managedAppNginxProperties {
		if err := setAppNginxProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppNginxProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	command := fmt.Sprintf("nginx:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))
	if value := d.Get(property).(string); value != "" {
		command += " " + shellescape.Quote(value)
	}
	return run(client, command).err
}

func readAppNginxProperties(app string, client *goph.Client) (map[string]string, bool, error) {
	if exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app))); exists.err != nil {
		if exists.status > 0 {
			return nil, false, nil
		}
		return nil, false, exists.err
	}
	res := run(client, fmt.Sprintf("nginx:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return nil, false, res.err
	}
	var report map[string]string
	if err := json.Unmarshal([]byte(res.stdout), &report); err != nil {
		return nil, false, fmt.Errorf("parsing nginx report for %q: %w", app, err)
	}
	return report, true, nil
}

func dokkuPropertyName(property string) string {
	return strings.ReplaceAll(property, "_", "-")
}
