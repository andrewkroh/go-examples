package filter

import (
	"gopkg.in/yaml.v3"
)

func Keys(n *yaml.Node, keys ...string) (changes int) {
	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}

	return filterKeys(n, keySet)
}

func filterKeys(n *yaml.Node, keySet map[string]struct{}) (numChanges int) {
	if n == nil {
		return 0
	}

	if n.Kind == yaml.MappingNode {
		nc := filterMappingNode(n, func(key *yaml.Node) bool {
			_, found := keySet[key.Value]
			return found
		})
		numChanges += nc
	}

	// Recurse
	for _, n := range n.Content {
		nc := filterKeys(n, keySet)
		numChanges += nc
	}

	return numChanges
}

func filterMappingNode(mappingNode *yaml.Node, predicate func(key *yaml.Node) bool) (numChanges int) {
	if mappingNode.Kind != yaml.MappingNode {
		panic("node is not a MappingNode")
	}

	content := make([]*yaml.Node, 0, len(mappingNode.Content))
	var key, value *yaml.Node
	for i, node := range mappingNode.Content {
		if i%2 == 0 {
			key = node
			continue
		}
		value = node

		// Filter (drop) pair when true.
		if predicate(key) {
			numChanges++
			continue
		}

		content = append(content, key, value)
	}

	mappingNode.Content = content
	return numChanges
}
