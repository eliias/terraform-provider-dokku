package provider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

func databaseServiceInitialNetworkSchema(service string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: fmt.Sprintf("Initial Docker network for the %s service container.", service),
	}
}

func databaseServicePostNetworkSchema(service, phase string) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Schema{
			Type:         schema.TypeString,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		Description: fmt.Sprintf("Docker networks attached to the %s service container %s.", service, phase),
	}
}

func serviceNetworkSet(d *schema.ResourceData, key string) []string {
	values := interfaceSliceToStrSlice(d.Get(key).(*schema.Set).List())
	sort.Strings(values)
	return values
}

func parseServiceNetworks(value string) []string {
	if value == "" || value == "-" {
		return nil
	}
	return splitNetworkValues(value)
}

func setDatabaseServiceNetworkProperty(client *goph.Client, service, name, property, value string) error {
	args := []string{
		shellescape.Quote(name),
		shellescape.Quote(property),
	}
	if value != "" {
		args = append(args, shellescape.Quote(value))
	}
	res := run(client, fmt.Sprintf("%s:set %s", service, strings.Join(args, " ")))
	return res.err
}

func databaseServiceNetworkValue(d *schema.ResourceData, key string) string {
	switch key {
	case "post_create_networks", "post_start_networks":
		return strings.Join(serviceNetworkSet(d, key), ",")
	default:
		return d.Get(key).(string)
	}
}

func updateDatabaseServiceNetworks(service *DokkuGenericService, d *schema.ResourceData, client *goph.Client) (bool, error) {
	properties := map[string]string{
		"initial_network":      "initial-network",
		"post_create_networks": "post-create-network",
		"post_start_networks":  "post-start-network",
	}
	changed := false
	for field, property := range properties {
		if !d.HasChange(field) {
			continue
		}
		changed = true
		if err := setDatabaseServiceNetworkProperty(client, service.CmdName, service.Name, property, databaseServiceNetworkValue(d, field)); err != nil {
			return false, err
		}
	}
	if !changed {
		return false, nil
	}

	oldStoppedValue, _ := d.GetChange("stopped")
	wasStopped := oldStoppedValue.(bool)
	if !wasStopped {
		if res := run(client, fmt.Sprintf("%s:stop %s", service.CmdName, shellescape.Quote(service.Name))); res.err != nil {
			return false, res.err
		}
	}
	if !service.Stopped {
		if res := run(client, fmt.Sprintf("%s:start %s", service.CmdName, shellescape.Quote(service.Name))); res.err != nil {
			return false, res.err
		}
	}
	return true, nil
}
