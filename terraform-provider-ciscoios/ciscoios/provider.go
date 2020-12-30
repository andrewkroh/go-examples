package ciscoios

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"ssh_host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CISCOIOS_SSH_HOST", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"ciscoios_acl": resourceCiscoIOSACL(),
		},
	}
}
