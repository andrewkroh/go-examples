package main

import (
	"github.com/andrewkroh/go-examples/terraform-provider-ciscoios/ciscoios"
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return ciscoios.Provider()
		},
	})
}
