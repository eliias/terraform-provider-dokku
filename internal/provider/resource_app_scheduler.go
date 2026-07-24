package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

type dokkuAppSchedulerReport struct {
	Selected         string `json:"selected"`
	Shell            string `json:"shell"`
	ComputedSelected string `json:"computed-selected"`
	ComputedShell    string `json:"computed-shell"`
}

type dokkuAppDockerLocalSchedulerReport struct {
	InitProcess                   string `json:"init-process"`
	ParallelScheduleCount         string `json:"parallel-schedule-count"`
	ComputedInitProcess           string `json:"computed-init-process"`
	ComputedParallelScheduleCount string `json:"computed-parallel-schedule-count"`
}

func resourceAppScheduler() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages explicit generic scheduler properties while reporting their inherited effective values.",
		CreateContext: resourceAppSchedulerCreate,
		ReadContext:   resourceAppSchedulerRead,
		UpdateContext: resourceAppSchedulerUpdate,
		DeleteContext: resourceAppSchedulerDelete,
		Schema: map[string]*schema.Schema{
			"app": appSettingsNameSchema("Dokku application whose scheduler properties are managed."),
			"selected": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explicit scheduler integration. Empty inherits the global selection and built-in docker-local default.",
			},
			"shell": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Explicit shell for dokku run and enter. Empty lets the scheduler select it.",
			},
			"effective_selected": computedStringSchema("Effective scheduler after global and built-in inheritance."),
			"effective_shell":    computedStringSchema("Effective shell after global inheritance."),
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func resourceAppSchedulerDockerLocal() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages docker-local scheduler properties while reporting their inherited effective values.",
		CreateContext: resourceAppSchedulerDockerLocalCreate,
		ReadContext:   resourceAppSchedulerDockerLocalRead,
		UpdateContext: resourceAppSchedulerDockerLocalUpdate,
		DeleteContext: resourceAppSchedulerDockerLocalDelete,
		Schema: map[string]*schema.Schema{
			"app": appSettingsNameSchema("Dokku application whose docker-local scheduler properties are managed."),
			"init_process": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"true", "false"}, false),
				Description:  "Explicit Docker init-process setting. Empty inherits the global value and built-in true default.",
			},
			"parallel_schedule_count": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "Maximum process types scheduled in parallel during deployment. Empty inherits the global value and built-in value of one.",
			},
			"effective_init_process": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Effective init-process setting.",
			},
			"effective_parallel_schedule_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Effective parallel schedule count.",
			},
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func appSettingsNameSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		ValidateFunc: validation.StringIsNotEmpty,
		Description:  description,
	}
}

func computedStringSchema(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Computed: true, Description: description}
}

func resourceAppSchedulerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllAppSchedulerProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppSchedulerRead(ctx, d, m)
}

func resourceAppSchedulerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := appSettingsResourceApp(d)
	report, exists, err := readAppSchedulerReport(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	d.SetId(app)
	d.Set("app", app)
	d.Set("selected", report.Selected)
	d.Set("shell", report.Shell)
	d.Set("effective_selected", report.ComputedSelected)
	d.Set("effective_shell", report.ComputedShell)
	return nil
}

func resourceAppSchedulerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range []string{"selected", "shell"} {
		if d.HasChange(property) {
			if err := setAppSchedulerProperty(d, property, client); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return resourceAppSchedulerRead(ctx, d, m)
}

func resourceAppSchedulerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	for _, property := range []string{"selected", "shell"} {
		if err := run(client, fmt.Sprintf("scheduler:set %s %s", shellescape.Quote(app), property)).err; err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId("")
	return nil
}

func setAllAppSchedulerProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range []string{"selected", "shell"} {
		if err := setAppSchedulerProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppSchedulerProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	command := fmt.Sprintf("scheduler:set %s %s", shellescape.Quote(app), property)
	if value := d.Get(property).(string); value != "" {
		command += " " + shellescape.Quote(value)
	}
	return run(client, command).err
}

func resourceAppSchedulerDockerLocalCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := setAllAppSchedulerDockerLocalProperties(d, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(d.Get("app").(string))
	return resourceAppSchedulerDockerLocalRead(ctx, d, m)
}

func resourceAppSchedulerDockerLocalRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := appSettingsResourceApp(d)
	report, exists, err := readAppSchedulerDockerLocalReport(app, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	d.SetId(app)
	d.Set("app", app)
	d.Set("init_process", report.InitProcess)
	if report.ParallelScheduleCount == "" {
		d.Set("parallel_schedule_count", nil)
	} else {
		value, err := strconv.Atoi(report.ParallelScheduleCount)
		if err != nil {
			return diag.Errorf("parsing docker-local parallel schedule count for %q: %v", app, err)
		}
		d.Set("parallel_schedule_count", value)
	}
	d.Set("effective_init_process", report.ComputedInitProcess == "true")
	effectiveCount, err := strconv.Atoi(report.ComputedParallelScheduleCount)
	if err != nil {
		return diag.Errorf("parsing effective docker-local parallel schedule count for %q: %v", app, err)
	}
	d.Set("effective_parallel_schedule_count", effectiveCount)
	return nil
}

func resourceAppSchedulerDockerLocalUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range []string{"init_process", "parallel_schedule_count"} {
		if d.HasChange(property) {
			if err := setAppSchedulerDockerLocalProperty(d, property, client); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	return resourceAppSchedulerDockerLocalRead(ctx, d, m)
}

func resourceAppSchedulerDockerLocalDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	for _, property := range []string{"init-process", "parallel-schedule-count"} {
		if err := run(client, fmt.Sprintf("scheduler-docker-local:set %s %s", shellescape.Quote(app), property)).err; err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId("")
	return nil
}

func setAllAppSchedulerDockerLocalProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range []string{"init_process", "parallel_schedule_count"} {
		if err := setAppSchedulerDockerLocalProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppSchedulerDockerLocalProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	command := fmt.Sprintf("scheduler-docker-local:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))
	if property == "init_process" {
		if value := d.Get(property).(string); value != "" {
			command += " " + value
		}
	} else if value, ok := d.GetOk(property); ok {
		command += " " + strconv.Itoa(value.(int))
	}
	return run(client, command).err
}

func appSettingsResourceApp(d *schema.ResourceData) string {
	if d.Id() != "" {
		return d.Id()
	}
	return d.Get("app").(string)
}

func readAppSchedulerReport(app string, client *goph.Client) (dokkuAppSchedulerReport, bool, error) {
	var report dokkuAppSchedulerReport
	exists, err := readAppJSONReport(app, "scheduler:report", &report, client)
	return report, exists, err
}

func readAppSchedulerDockerLocalReport(app string, client *goph.Client) (dokkuAppDockerLocalSchedulerReport, bool, error) {
	var report dokkuAppDockerLocalSchedulerReport
	exists, err := readAppJSONReport(app, "scheduler-docker-local:report", &report, client)
	return report, exists, err
}

func readAppJSONReport(app, command string, destination interface{}, client *goph.Client) (bool, error) {
	if exists := run(client, fmt.Sprintf("apps:exists %s", shellescape.Quote(app))); exists.err != nil {
		if exists.status > 0 {
			return false, nil
		}
		return false, exists.err
	}
	res := run(client, fmt.Sprintf("%s %s --format json", command, shellescape.Quote(app)))
	if res.err != nil {
		return false, res.err
	}
	if err := json.Unmarshal([]byte(res.stdout), destination); err != nil {
		return false, fmt.Errorf("parsing %s JSON report for %q: %w", command, app, err)
	}
	return true, nil
}
