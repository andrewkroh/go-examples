terraform {
  required_providers {
    ciscoios = {
      version = "~> 0.0.1"
      source  = "crowbird.com/andrewkroh/ciscoios"
    }
  }
}

provider "ciscoios" {
  ssh_host = "127.0.0.1"
}

resource "ciscoios_acl" "guest_acl_in" {
  name = "guest_acl_in"
  rule {
    remarks = ["Allow responses for established TCP connections."]
    protocol = "tcp"
    destination_port = "gt 1023"
    established = true
  }
}
