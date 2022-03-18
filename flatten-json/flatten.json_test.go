package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

const sampleDoc = `{
    "agent": {
      "hostname": "692bc2a32631",
      "name": "compute01.hc.foo.local.example.com",
      "id": "814bb722-1e0c-4fd0-a661-8e7cb3087445",
      "ephemeral_id": "3eed18c0-fdd6-4bd2-8122-cedd926f73a2",
      "type": "filebeat",
      "version": "7.16.2"
    },
    "destination": {
      "port": 39960,
      "ip": "10.100.18.2",
      "domain": "smarthings-hub.iot.foo.local.example.com",
      "locality": "internal",
      "mac": "04:18:d6:f1:2c:20"
    },
    "source": {
      "geo": {
        "continent_name": "North America",
        "region_iso_code": "US-OH",
        "city_name": "Columbus",
        "country_iso_code": "US",
        "country_name": "United States",
        "region_name": "Ohio",
        "location": {
          "lon": -83.0235,
          "lat": 39.9653
        }
      },
      "as": {
        "number": 16509,
        "organization": {
          "name": "Amazon.com, Inc."
        }
      },
      "port": 443,
      "bytes": 513,
      "ip": "13.59.192.161",
      "locality": "external",
      "mac": "e8:6f:f2:31:21:e0",
      "packets": 7
    },
    "network": {
      "community_id": "1:uQKAlB8cUbkc/+W1pEuU0C+TLVE=",
      "bytes": 513,
      "name": [
        "foo-iot"
      ],
      "transport": "tcp",
      "packets": 7,
      "iana_number": "6",
      "direction": "inbound"
    },
    "tags": [
      "foo",
      "forwarded",
      "netflow",
      "udp_7104"
    ],
    "observer": {
      "geo": {
        "name": "foo"
      },
      "hostname": "compute01.hc.foo.local.example.com",
      "product": "Filebeat",
      "vendor": "Elastic",
      "ip": [
        "10.100.16.34"
      ],
      "name": "compute01-hc-foo-local-example-com"
    },
    "input": {
      "type": "netflow"
    },
    "netflow": {
      "protocol_identifier": 6,
      "packet_delta_count": 7,
      "vlan_id": 0,
      "source_mac_address": "e8:6f:f2:31:21:e0",
      "flow_start_sys_up_time": 2397772818,
      "egress_interface": 3,
      "octet_delta_count": 513,
      "type": "netflow_flow",
      "destination_ipv4_address": "10.100.18.2",
      "source_ipv4_address": "13.59.192.161",
      "delta_flow_count": 0,
      "exporter": {
        "uptime_millis": 2397846583,
        "address": "10.100.30.6:47408",
        "source_id": 0,
        "version": 9,
        "timestamp": "2022-03-18T15:13:24.000Z"
      },
      "tcp_control_bits": 24,
      "ip_class_of_service": 0,
      "ip_version": 4,
      "flow_direction": 0,
      "mpls_label_stack_length": 2,
      "ingress_interface": 2,
      "destination_mac_address": "04:18:d6:f1:2c:20",
      "flow_end_sys_up_time": 2397842270,
      "source_transport_port": 443,
      "destination_transport_port": 39960
    },
    "@timestamp": "2022-03-18T15:13:24.000Z",
    "ecs": {
      "version": "1.12.0"
    },
    "related": {
      "ip": [
        "13.59.192.161",
        "10.100.18.2"
      ]
    },
    "data_stream": {
      "type": "logs",
      "dataset": "netflow.log"
    },
    "event": {
      "duration": 69452000000,
      "agent_id_status": "auth_metadata_missing",
      "ingested": "2022-03-18T15:13:24Z",
      "created": "2022-03-18T15:13:24.080Z",
      "kind": "event",
      "start": "2022-03-18T15:12:10.235Z",
      "action": "netflow_flow",
      "end": "2022-03-18T15:13:19.687Z",
      "category": [
        "network_traffic",
        "network"
      ],
      "type": [
        "connection"
      ]
    },
    "flow": {
      "locality": "external",
      "id": "13l__e8srZI"
    }
  }`

func TestFlatten(t *testing.T) {
	obj, err := ReadObject(bytes.NewBufferString(sampleDoc))
	require.NoError(t, err)

	flat := Flatten(obj)

	jsonStr, err := toJSONString(flat, true)
	require.NoError(t, err)
	t.Log(jsonStr)

	for _, k := range KeyList(flat) {
		t.Log(k)
	}
}

func toJSONString(m map[string]interface{}, pretty bool) (string, error) {
	buf := new(bytes.Buffer)
	if err := ToJSON(m, pretty, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
