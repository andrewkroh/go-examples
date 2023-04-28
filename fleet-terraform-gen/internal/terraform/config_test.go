package terraform_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andrewkroh/go-examples/fleet-terraform-gen/internal/terraform"
)

func TestMarshalJSON(t *testing.T) {
	data := terraform.File{
		Variables: map[string]terraform.Variable{
			"foo": {
				Type:        "string",
				Description: "Configures the bar.",
			},
		},
		Modules: map[string]terraform.Module{
			"package_policy_github_issues": {
				Source: "../../fleet_package_policy",
				Params: map[string]any{
					"agent_policy_id": "ebe2efde-b965-4df3-9d2c-1dc8b808fe72",
				},
			},
		},
		Outputs: map[string]terraform.Output{
			"agent_policy_id": {
				Description: "Agent policy ID",
				Sensitive:   terraform.Ptr(true),
				Value:       "${module.fleet_agent_policy.id}",
			},
		},
	}

	expected := `
{
  "variable": {
    "foo": {
      "type": "string",
      "description": "Configures the bar."
    }
  },
  "output": {
    "agent_policy_id": {
      "description": "Agent policy ID",
      "sensitive": true,
      "value": "${module.fleet_agent_policy.id}"
    }
  },
  "module": {
    "package_policy_github_issues": {
      "agent_policy_id": "ebe2efde-b965-4df3-9d2c-1dc8b808fe72",
      "source": "../../fleet_package_policy"
    }
  }
}

`
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		t.Fatal(err)
	}

	assert.JSONEq(t, expected, buf.String())
}
