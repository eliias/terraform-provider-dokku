package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/melbahja/goph"
)

type dokkuPluginCapability struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	Core        bool   `json:"core"`
	Description string `json:"description"`
	SourceURL   string `json:"source_url"`
}

func dataSourceCapabilities() *schema.Resource {
	return &schema.Resource{
		Description: "Reports the Dokku version and enabled plugin capabilities available on the configured host.",
		ReadContext: dataSourceCapabilitiesRead,
		Schema: map[string]*schema.Schema{
			"dokku_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Dokku version reported by the configured host.",
			},
			"tested_version": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this provider declares the host Dokku version tested.",
			},
			"plugins": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Enabled Dokku plugins and their reported metadata.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"name":        {Type: schema.TypeString, Computed: true},
					"version":     {Type: schema.TypeString, Computed: true},
					"core":        {Type: schema.TypeBool, Computed: true},
					"description": {Type: schema.TypeString, Computed: true},
					"source_url":  {Type: schema.TypeString, Computed: true},
				}},
			},
			"builders":   capabilityNameSetSchema("Enabled builder integration names without the builder- prefix."),
			"schedulers": capabilityNameSetSchema("Enabled scheduler integration names without the scheduler- prefix."),
			"proxies":    capabilityNameSetSchema("Enabled proxy integration names without the -vhosts suffix."),
		},
	}
}

func capabilityNameSetSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Description: description,
	}
}

func dataSourceCapabilitiesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	res := run(m.(*goph.Client), "plugin:list --enabled --format json")
	if res.err != nil {
		return diag.FromErr(res.err)
	}

	plugins, err := parseDokkuPluginCapabilities(res.stdout)
	if err != nil {
		return diag.FromErr(err)
	}
	builders, schedulers, proxies := classifyDokkuPluginCapabilities(plugins)

	version := DOKKU_VERSION.String()
	d.SetId("host")
	if err := d.Set("dokku_version", version); err != nil {
		return diag.FromErr(err)
	}
	compat, err := semver.ParseRange(testedDokkuVersions)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("tested_version", compat(DOKKU_VERSION)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("plugins", dokkuPluginCapabilitiesToInterfaces(plugins)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("builders", builders); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("schedulers", schedulers); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("proxies", proxies); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func parseDokkuPluginCapabilities(stdout string) ([]dokkuPluginCapability, error) {
	var plugins []dokkuPluginCapability
	if err := json.Unmarshal([]byte(stdout), &plugins); err != nil {
		return nil, fmt.Errorf("parsing enabled Dokku plugin list: %w", err)
	}
	enabled := plugins[:0]
	for _, plugin := range plugins {
		if plugin.Enabled {
			enabled = append(enabled, plugin)
		}
	}
	sort.Slice(enabled, func(i, j int) bool { return enabled[i].Name < enabled[j].Name })
	return enabled, nil
}

func classifyDokkuPluginCapabilities(plugins []dokkuPluginCapability) ([]string, []string, []string) {
	var builders, schedulers, proxies []string
	for _, plugin := range plugins {
		switch {
		case strings.HasPrefix(plugin.Name, "builder-"):
			builders = append(builders, strings.TrimPrefix(plugin.Name, "builder-"))
		case strings.HasPrefix(plugin.Name, "scheduler-"):
			schedulers = append(schedulers, strings.TrimPrefix(plugin.Name, "scheduler-"))
		case strings.HasSuffix(plugin.Name, "-vhosts"):
			proxies = append(proxies, strings.TrimSuffix(plugin.Name, "-vhosts"))
		}
	}
	sort.Strings(builders)
	sort.Strings(schedulers)
	sort.Strings(proxies)
	return builders, schedulers, proxies
}

func dokkuPluginCapabilitiesToInterfaces(plugins []dokkuPluginCapability) []interface{} {
	values := make([]interface{}, 0, len(plugins))
	for _, plugin := range plugins {
		values = append(values, map[string]interface{}{
			"name":        plugin.Name,
			"version":     plugin.Version,
			"core":        plugin.Core,
			"description": plugin.Description,
			"source_url":  plugin.SourceURL,
		})
	}
	return values
}
