MD := go run moul.io/mdtable@v1.0.0 csv
OCSF := go run ./cmd/ocsf-fields

.PHONY: all
all: csv readme

.PHONY: readme
readme: csv
	echo "# OCSF 1.3" > README.md

	echo >> README.md
	echo '## All Fields (using depth 0)' >> README.md
	echo >> README.md
	echo 'total count = ' >> README.md
	$(OCSF) -f -c| jq 'length'  >> README.md

	echo >> README.md
	echo '```mermaid' >> README.md
	echo 'pie showData title Fields by type' >> README.md
	cut -d ',' -f2 ocsf_1_3.csv | sort | uniq -c | awk '{printf("  \"%s\": %d\n", $$2, $$1)}' >> README.md
	echo '```' >> README.md
	echo >> README.md
	echo '(`null` is where traversal ended due to circular references.)' >> README.md


	echo >> README.md
	echo '## Field count by class (using depth 0)' >> README.md
	$(OCSF) -f | jq 'map_values(length)' | yq -o csv 'toentries' | $(MD) >> README.md

	echo >> README.md
	echo '## Field count by class (using depth 1)' >> README.md
	$(OCSF) -f -d 1 | jq 'map_values(length)' | yq -o csv 'toentries' | $(MD) >> README.md

.PHONY: csv
csv:
	$(OCSF) -f -c | jq 'map({(.name): .type}) | add' | yq -o csv 'toentries' > ocsf_1_3.csv
