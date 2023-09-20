# ecs-update

This is a tool for automating modifications to Fleet packages.
It began as a tool to change the `ecs.version`, but now
can be used for other batch modifications like setting `format_version`.

You should have your `elastic-package stack up` environment running
and configured because the tool will execute `elastic-package test pipeline -g`.

After modifying a package, it will commit the results. It writes one commit
per package. If there is a failure, then it will not commit the result. It
will leave the modified files in place and move onto the next package.

The commit messages will contain a detailed summary of the changes applied.
It will also include a `[git-generate]` command that can be used with
https://pkg.go.dev/rsc.io/rf/git-generate to automatically fix merge conflicts
for the generated commits. It also gives full traceability as to how the
changes were made.

## Installation

`go install github.com/andrewkroh/go-examples/ecs-update@main`

## Usage examples

Update ECS versions.

```sh
ecs-update \
  -pr 999999999 \
  -ecs-git-ref v8.10.0 \
  -ecs-version 8.10.0 \
  -owner elastic/security-external-integrations \
  packages/*
```

-----

Update `format_version` to 3.0.0. And because 3.0.0 requires fixing dotted
YAML keys we'll add `-fix-dotted-yaml-keys` to update the `conditions` attribute
of package manifests. Also new in 3.0.0 is the `owner.type` field, so we'll
add that to packages if it does not exist already via `-add-owner-type`.

```sh
ecs-update \
  -format-version=3.0.0 \
  -fix-dotted-yaml-keys \
  -add-owner-type \
  -owner elastic/security-external-integrations
```

## Result report

When the tool exits, it reports its exit code as non-zero on failure. It will also
write a status message like

> Completed. No errors. Details written to /var/folders/tt/8k1x54gn2tv2fkhr9hk41z_w0000gn/T/ecs-update-result.json

or

> Interrupted. Failed. Details written to /var/folders/tt/8k1x54gn2tv2fkhr9hk41z_w0000gn/T/ecs-update-result.json`

The linked file contains details about each updated package. This shows a package that
was no modified.

```json
[
  {
    "package": "1password",
    "changed": false,
    "failed": false
  }
]
```

This shows a package that was modified and includes the stderr and stdout of
all sub-commands that were executed.

```json
[
  {
    "package": "1password",
    "changed": true,
    "failed": false,
    "stdout": "--- Test results for package: 1password - START ---\n<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites>\n  <testsuite name=\"pipeline\" tests=\"3\">\n    <!--test suite for pipeline tests-->\n    <testcase name=\"pipeline test: test-auditevents.json\" classname=\"1password.audit_events\" time=\"0.003402917\"></testcase>\n    <testcase name=\"pipeline test: test-itemusages.json\" classname=\"1password.item_usages\" time=\"0.002245292\"></testcase>\n    <testcase name=\"pipeline test: test-signinattempts.json\" classname=\"1password.signin_attempts\" time=\"0.002274333\"></testcase>\n  </testsuite>\n</testsuites>\n--- Test results for package: 1password - END   ---\nDone\n",
    "stderr": "2023/09/20 13:21:52  INFO New version is available - v0.87.1. Download from: https://github.com/elastic/elastic-package/releases/tag/v0.87.1\nFormat the package\nDone\n2023/09/20 13:21:52  INFO New version is available - v0.87.1. Download from: https://github.com/elastic/elastic-package/releases/tag/v0.87.1\nBuild the package\nREADME.md file rendered: /Users/akroh/code/elastic/integrations/packages/1password/docs/README.md\n2023/09/20 13:21:52  INFO License text found in \"/Users/akroh/code/elastic/integrations/LICENSE.txt\" will be included in package\n2023/09/20 13:21:52 Warning: conditions.kibana.version must be ^8.10.0 or greater to include saved object tags file: kibana/tags.yml\nPackage built: /Users/akroh/code/elastic/integrations/build/packages/1password-1.20.0.zip\nDone\n2023/09/20 13:21:52  INFO New version is available - v0.87.1. Download from: https://github.com/elastic/elastic-package/releases/tag/v0.87.1\nRun pipeline tests for the package\n"
  }
]
```
