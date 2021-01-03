package ciscoios

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"ssh_address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CISCOIOS_SSH_ADDRESS", nil),
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CISCOIOS_USERNAME", nil),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CISCOIOS_PASSWORD", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"ciscoios_acl": resourceCiscoIOSACL(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Address:  d.Get("ssh_address").(string),
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
	}

	return config.NewClient()
}
