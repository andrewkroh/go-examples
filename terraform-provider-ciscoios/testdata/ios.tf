terraform {
  required_providers {
    ciscoios = {
      version = "~> 0.0.1"
      source  = "crowbird.com/andrewkroh/ciscoios"
    }
  }
}

provider "ciscoios" {
  ssh_address = "mock://127.0.0.1:22"
  username = "foo"
  password = "bar"
}

resource "ciscoios_acl" "guest_acl_in" {
  name = "guest_acl_in"
  rule {
    remarks = ["Allow responses for established TCP connections."]
    protocol = "tcp"
    destination_port = "gt 1023"
    established = true
  }

  rule {
    remarks = ["Allow pings."]
    protocol = "icmp"
  }
}
