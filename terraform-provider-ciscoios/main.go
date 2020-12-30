package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"

	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/ciscoios"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return ciscoios.Provider()
		},
	})
}
