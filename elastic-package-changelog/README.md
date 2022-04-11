# elastic-package-changelog

This is a command line tool for adding new entries to the changelog.yml file
in Elastic Fleet integrations. The version contained in the manifest.yml is
updated with the new version.

It determines the new version automatically based on the change type.

- `bugfix` - Bump patch.
- `enhancement` - Bump minor.
- `breaking-change` - Bump major.

If a pull request number is not specific via `--pr` then a placeholder is
added (`{{ PULL_REQUEST_NUMBER }}`). After you know the PR number you can
find/replace that value.

# Installation

`go install github.com/andrewkroh/go-examples/elastic-package-changelog@main`

# Usage

```
$ elastic-package-changelog add-next -h
Add a change under a new (next) version.

Usage:
  elastic-package-changelog add-next [flags]

Flags:
  -c, --changelog string     Changelog file to modify. (default "changelog.yml")
  -d, --description string   Description of change (use a proper sentence). Target audience is end users.
  -h, --help                 help for add-next
  -m, --manifest string      Manifest file to modify. (default "manifest.yml")
      --pr int               Pull request number.
      --type string          Change type (enhancement, bugfix, breaking-change).
```

# Example

```
cd elastic/integrations/packages/aws
elastic-package-changelog add-next --type bugfix --description "Add field definitions for ECS event.created and event.duration." --pr=3781
```

The changelog.yml and manifest.yml files are modified in place.

```diff
diff --git a/packages/aws/changelog.yml b/packages/aws/changelog.yml
index 04acf1e4f..682c175b1 100644
--- a/packages/aws/changelog.yml
+++ b/packages/aws/changelog.yml
@@ -1,4 +1,9 @@
 # newer versions go on top
+- version: "1.14.4"
+  changes:
+    - description: Add field definitions for ECS event.created and event.duration.
+      type: bugfix
+      link: https://github.com/elastic/integrations/pull/3781
 - version: "1.14.3"
   changes:
     - description: Add new pattern to VPC Flow logs including all 29 v5 fields
diff --git a/packages/aws/manifest.yml b/packages/aws/manifest.yml
index 9e986a86e..adf047218 100644
--- a/packages/aws/manifest.yml
+++ b/packages/aws/manifest.yml
@@ -1,7 +1,7 @@
 format_version: 1.0.0
 name: aws
 title: AWS
-version: 1.14.3
+version: "1.14.4"
 license: basic
 description: Collect logs and metrics from Amazon Web Services with Elastic Agent.
 type: integration
```

# Future Addition Ideas

- Add flags to control what part of the version increment (e.g. `--major`,
`--minor`, `--patch`). For experimental and beta packages this may be necessary.
- Add the ability to add to the current release (e.g. `add-current`).
