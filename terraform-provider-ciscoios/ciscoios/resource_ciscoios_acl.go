package ciscoios

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pkg/errors"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/client"
)

var aclRule = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"permit": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},

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

		"established": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},

		"log": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},

		"remarks": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
	},
}

func resourceCiscoIOSACL() *schema.Resource {
	return &schema.Resource{
		Create: resourceCiscoIOSACLCreate,
		Read:   resourceCiscoIOSACLRead,
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

func resourceCiscoIOSACLCreate(d *schema.ResourceData, meta interface{}) error {
	cl := meta.(*client.Client)
	acls, err := cl.ACLs()
	if err != nil {
		return err
	}

	id, err := client.FreeExtendedAccessListID(acls)
	if err != nil {
		return err
	}

	acl := client.AccessList{
		ID: id,
	}
	d.SetId(acl.ID)

	for _, ifc := range d.Get("rule").([]interface{}) {
		ruleMap := ifc.(map[string]interface{})

		if remarks, ok := ruleMap["remarks"].([]interface{}); ok && len(remarks) > 0 {
			for _, remark := range remarks {
				acl.Rules = append(acl.Rules, client.AccessListEntry{
					Remark: remark.(string),
				})
			}
		}

		ale := client.AccessListEntry{}
		ale.Permit = ruleMap["permit"].(bool)
		ale.Protocol = ruleMap["protocol"].(string)
		ale.Source = ruleMap["source"].(string)
		ale.SourcePort = ruleMap["source_port"].(string)
		ale.Destination = ruleMap["destination"].(string)
		ale.DestinationPort = ruleMap["destination_port"].(string)
		ale.Established = ruleMap["established"].(bool)
		acl.Rules = append(acl.Rules, ale)
	}

	log.Printf("Create ACL: %v", spew.Sdump(acl))
	if err = cl.CreateACL(acl); err != nil {
		return err
	}

	return resourceCiscoIOSACLRead(d, meta)
}

func resourceCiscoIOSACLRead(d *schema.ResourceData, meta interface{}) error {
	cl := meta.(*client.Client)

	acls, err := cl.ACLs()
	if err != nil {
		return err
	}

	var acl *client.AccessList
	for _, item := range acls {
		if item.ID == d.Id() {
			acl = &item
			break
		}
	}

	// ACL no longer exists.
	if acl == nil {
		d.SetId("")
		return nil
	}

	var rules []interface{}
	for _, r := range acl.Rules {
		if r.Remark != "" {
			rules = append(rules, map[string]interface{}{
				"remark": r.Remark,
			})
			continue
		}

		rule := map[string]interface{}{
			"permit":           r.Permit,
			"protocol":         r.Protocol,
			"source":           r.Source,
			"source_port":      r.SourcePort,
			"destination":      r.Destination,
			"destination_port": r.DestinationPort,
			"established":      r.Established,
			"log":              r.Log,
		}
		rules = append(rules, rule)
	}

	d.Set("rule", rules)
	return nil
}

func resourceCiscoIOSACLDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("Delete ACL with ID=%v", d.Id())
	cl := meta.(*client.Client)
	return cl.DeleteACL(d.Id())
}
