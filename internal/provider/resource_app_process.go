package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

var managedAppProcessProperties = []string{
	"procfile_path",
	"restart_policy",
	"skip_deploy",
	"start_cmd",
	"dockerfile_start_cmd",
	"stop_timeout_seconds",
	"restore",
}

type dokkuAppProcessReport struct {
	ProcfilePath               string `json:"procfile-path"`
	RestartPolicy              string `json:"restart-policy"`
	Restore                    string `json:"restore"`
	SkipDeploy                 string `json:"skip-deploy"`
	StartCmd                   string `json:"start-cmd"`
	DockerfileStartCmd         string `json:"dockerfile-start-cmd"`
	StopTimeoutSeconds         string `json:"stop-timeout-seconds"`
	ComputedProcfilePath       string `json:"computed-procfile-path"`
	ComputedRestartPolicy      string `json:"computed-restart-policy"`
	ComputedSkipDeploy         string `json:"computed-skip-deploy"`
	ComputedStartCmd           string `json:"computed-start-cmd"`
	ComputedDockerfileStartCmd string `json:"computed-dockerfile-start-cmd"`
	ComputedStopTimeoutSeconds string `json:"computed-stop-timeout-seconds"`
	CanScale                   string `json:"can-scale"`
	Deployed                   string `json:"deployed"`
	Running                    string `json:"running"`
	Processes                  string `json:"processes"`
}

type dokkuAppProcessScale struct {
	ProcessType string `json:"process_type"`
	Quantity    int    `json:"quantity"`
}

func resourceAppProcess() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages persistent process properties and exact process scale for a Dokku application.",
		CreateContext: resourceAppProcessCreate,
		ReadContext:   resourceAppProcessRead,
		UpdateContext: resourceAppProcessUpdate,
		DeleteContext: resourceAppProcessDelete,
		Schema: map[string]*schema.Schema{
			"app":            appSettingsNameSchema("Dokku application whose process settings are managed."),
			"procfile_path":  processOptionalStringSchema("Explicit Procfile path relative to the build root."),
			"restart_policy": processOptionalStringSchema("Explicit Docker restart policy. Changing it requires a later rebuild to affect existing containers."),
			"skip_deploy": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"true", "false"}, false),
				Description:  "Explicit deploy-phase skip setting. Empty inherits.",
			},
			"start_cmd":            processOptionalStringSchema("Explicit buildpack application start command."),
			"dockerfile_start_cmd": processOptionalStringSchema("Explicit Dockerfile application start command."),
			"stop_timeout_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "Explicit seconds Docker waits before forcefully stopping a container. Empty inherits.",
			},
			"restore": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Whether Dokku restores the app after host reboot.",
			},
			"scale": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "Exact process quantities, including zero-count process types. Changes invoke ps:scale immediately.",
				Elem: &schema.Schema{
					Type:         schema.TypeInt,
					ValidateFunc: validation.IntAtLeast(0),
				},
			},
			"effective_procfile_path":        computedStringSchema("Effective Procfile path."),
			"effective_restart_policy":       computedStringSchema("Effective Docker restart policy."),
			"effective_skip_deploy":          computedBoolSchema("Effective deploy-phase skip setting."),
			"effective_start_cmd":            computedStringSchema("Effective buildpack start command."),
			"effective_dockerfile_start_cmd": computedStringSchema("Effective Dockerfile start command."),
			"effective_stop_timeout_seconds": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Effective stop timeout.",
			},
			"can_scale": computedBoolSchema("Whether Dokku reports that this app supports horizontal scaling."),
			"deployed":  computedBoolSchema("Whether the app has a successful deployment."),
			"running":   computedBoolSchema("Whether any app process is running."),
			"processes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total scaled process count.",
			},
		},
		Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
	}
}

func processOptionalStringSchema(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Optional: true, Description: description}
}

func computedBoolSchema(description string) *schema.Schema {
	return &schema.Schema{Type: schema.TypeBool, Computed: true, Description: description}
}

func resourceAppProcessCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	if err := setAllAppProcessProperties(d, client); err != nil {
		return diag.FromErr(err)
	}
	if _, configured := d.GetOk("scale"); configured {
		if err := setAppProcessScale(d.Get("app").(string), processScaleFromResourceData(d), client); err != nil {
			return diag.FromErr(err)
		}
	}
	d.SetId(d.Get("app").(string))
	return resourceAppProcessRead(ctx, d, m)
}

func resourceAppProcessRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	app := appSettingsResourceApp(d)
	client := m.(*goph.Client)
	report, exists, err := readAppProcessReport(app, client)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}
	scale, err := readAppProcessScale(app, client)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(app)
	d.Set("app", app)
	d.Set("procfile_path", report.ProcfilePath)
	d.Set("restart_policy", report.RestartPolicy)
	d.Set("restore", report.Restore == "true")
	d.Set("skip_deploy", report.SkipDeploy)
	d.Set("start_cmd", report.StartCmd)
	d.Set("dockerfile_start_cmd", report.DockerfileStartCmd)
	setOptionalIntString(d, "stop_timeout_seconds", report.StopTimeoutSeconds)
	d.Set("scale", processScaleToMap(scale))
	d.Set("effective_procfile_path", report.ComputedProcfilePath)
	d.Set("effective_restart_policy", report.ComputedRestartPolicy)
	d.Set("effective_skip_deploy", report.ComputedSkipDeploy == "true")
	d.Set("effective_start_cmd", report.ComputedStartCmd)
	d.Set("effective_dockerfile_start_cmd", report.ComputedDockerfileStartCmd)
	setRequiredIntString(d, "effective_stop_timeout_seconds", report.ComputedStopTimeoutSeconds)
	d.Set("can_scale", report.CanScale == "true")
	d.Set("deployed", report.Deployed == "true")
	d.Set("running", report.Running == "true")
	setRequiredIntString(d, "processes", report.Processes)
	return nil
}

func resourceAppProcessUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	for _, property := range managedAppProcessProperties {
		if d.HasChange(property) {
			if err := setAppProcessProperty(d, property, client); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	if d.HasChange("scale") {
		if err := setAppProcessScale(d.Get("app").(string), processScaleFromResourceData(d), client); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceAppProcessRead(ctx, d, m)
}

func resourceAppProcessDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*goph.Client)
	app := d.Get("app").(string)
	for _, property := range managedAppProcessProperties {
		if err := run(client, fmt.Sprintf("ps:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))).err; err != nil {
			return diag.FromErr(err)
		}
	}
	// Scale is intentionally retained when Terraform relinquishes ownership.
	d.SetId("")
	return nil
}

func setAllAppProcessProperties(d *schema.ResourceData, client *goph.Client) error {
	for _, property := range managedAppProcessProperties {
		if property == "restore" {
			if _, configured := d.GetOkExists(property); !configured {
				continue
			}
		}
		if err := setAppProcessProperty(d, property, client); err != nil {
			return err
		}
	}
	return nil
}

func setAppProcessProperty(d *schema.ResourceData, property string, client *goph.Client) error {
	app := d.Get("app").(string)
	command := fmt.Sprintf("ps:set %s %s", shellescape.Quote(app), dokkuPropertyName(property))
	switch property {
	case "restore":
		command += " " + strconv.FormatBool(d.Get(property).(bool))
	case "stop_timeout_seconds":
		if value, ok := d.GetOk(property); ok {
			command += " " + strconv.Itoa(value.(int))
		}
	default:
		if value := d.Get(property).(string); value != "" {
			command += " " + shellescape.Quote(value)
		}
	}
	return run(client, command).err
}

func setAppProcessScale(app string, scale map[string]int, client *goph.Client) error {
	processTypes := make([]string, 0, len(scale))
	for processType := range scale {
		processTypes = append(processTypes, processType)
	}
	sort.Strings(processTypes)
	parts := make([]string, 0, len(scale))
	for _, processType := range processTypes {
		parts = append(parts, fmt.Sprintf("%s=%d", processType, scale[processType]))
	}
	if len(parts) == 0 {
		return fmt.Errorf("scale for %q cannot be empty", app)
	}
	command := fmt.Sprintf("ps:scale %s", shellescape.Quote(app))
	for _, part := range parts {
		command += " " + shellescape.Quote(part)
	}
	return run(client, command).err
}

func processScaleFromResourceData(d *schema.ResourceData) map[string]int {
	scale := make(map[string]int)
	for processType, value := range d.Get("scale").(map[string]interface{}) {
		scale[processType] = value.(int)
	}
	return scale
}

func processScaleToMap(scale []dokkuAppProcessScale) map[string]int {
	values := make(map[string]int, len(scale))
	for _, item := range scale {
		values[item.ProcessType] = item.Quantity
	}
	return values
}

func readAppProcessReport(app string, client *goph.Client) (dokkuAppProcessReport, bool, error) {
	var report dokkuAppProcessReport
	exists, err := readAppJSONReport(app, "ps:report", &report, client)
	return report, exists, err
}

func readAppProcessScale(app string, client *goph.Client) ([]dokkuAppProcessScale, error) {
	res := run(client, fmt.Sprintf("ps:scale %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return nil, res.err
	}
	var scale []dokkuAppProcessScale
	if err := json.Unmarshal([]byte(res.stdout), &scale); err != nil {
		return nil, fmt.Errorf("parsing process scale for %q: %w", app, err)
	}
	return scale, nil
}

func setOptionalIntString(d *schema.ResourceData, key, value string) {
	if value == "" {
		d.Set(key, nil)
		return
	}
	setRequiredIntString(d, key, value)
}

func setRequiredIntString(d *schema.ResourceData, key, value string) {
	parsed, err := strconv.Atoi(value)
	if err == nil {
		d.Set(key, parsed)
	}
}
