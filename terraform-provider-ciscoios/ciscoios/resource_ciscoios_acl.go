package ciscoios

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
)

var aclRule = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"protocol": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Default:  "ip",
			ValidateFunc: func(i interface{}, s string) (warnings []string, errs []error) {
				switch s {
				case "ip", "icmp", "tcp", "udp":
					return nil, []error{errors.Errorf("invalid value %v", s)}
				}
				return nil, nil
			},
		},

		"source": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Default:  "any",
		},

		"source_port": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"destination": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Default:  "any",
		},

		"destination_port": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},

		"log": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},

		"permit": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},

		"established": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},

		"remarks": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},

		"id": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
	},
}

func resourceCiscoIOSACL() *schema.Resource {
	return &schema.Resource{
		Create: resourceCiscoASAACLCreate,
		// TODO:
		Read: resourceCiscoIOSACLRead,
		//Update: resourceCiscoASAACLUpdate,
		Delete: resourceCiscoIOSACLDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"rule": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     aclRule,
				ForceNew: true,
			},
		},
	}
}

func resourceCiscoASAACLCreate(d *schema.ResourceData, meta interface{}) error {
	// We need to set this upfront in order to be able to save a partial state
	d.SetId(d.Get("name").(string))

	// Create all rules that are configured
	if nrs, ok := d.Get("rule").([]interface{}); ok && len(nrs) > 0 {
		log.Printf("%#v", nrs)

	}

	return nil
}

func resourceCiscoIOSACLRead(d *schema.ResourceData, meta interface{}) error {
	d.Set("rule", []interface{}{
		map[string]interface{}{
			"source": "any",
		},
	})

	return nil
}

func resourceCiscoIOSACLDelete(d *schema.ResourceData, meta interface{}) error {
	// Delete all rules
	if ors, ok := d.Get("rule").([]interface{}); ok && len(ors) > 0 {
		// Create an additional list with all the existing rules. Each rule that is
		// successfully deleted will be removed from this list, leaving only rules that
		// could not be deleted properly and should be saved in the state.
		rules := append([]interface{}(nil), ors...)

		log.Printf("Delete ID %v, %#v", d.Id(), &rules)
	}

	return nil
}
