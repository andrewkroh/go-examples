-- name: InsertManifest :exec
INSERT INTO manifests (dir_name, name, title, version, release, description, type, format_version, license, kibana_version, elastic_subscription, source_license, owner_github, owner_type)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertManifestIcon :exec
INSERT INTO manifest_icons (manifest_dir_name, src, title, size, type, dark_mode)
VALUES (?, ?, ?, ?, ?, ?);

-- name: InsertManifestCategory :exec
INSERT INTO manifest_categories (manifest_dir_name, category)
VALUES (?, ?);

-- name: InsertManifestScreenshot :exec
INSERT INTO manifest_screenshots (manifest_dir_name, src, title, size, type)
VALUES (?, ?, ?, ?, ?);

-- name: InsertManifestVar :exec
INSERT INTO manifest_vars (manifest_dir_name, name, default_value, description, type, title, multi, required, secret, show_user)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertManifestPolicyTemplate :exec
INSERT INTO manifest_policy_templates (manifest_dir_name, name, title, description)
VALUES (?, ?, ?, ?);

-- name: InsertManifestVarOption :exec
INSERT INTO manifest_var_options (manifest_dir_name, var_name, value, text)
VALUES (?, ?, ?, ?);

-- name: InsertManifestPolicyTemplateCategory :exec
INSERT INTO manifest_policy_template_categories (manifest_dir_name, policy_template_name, category)
VALUES (?, ?, ?);

-- name: InsertManifestPolicyTemplateDataStream :exec
INSERT INTO manifest_policy_template_data_streams (manifest_dir_name, policy_template_name, data_stream)
VALUES (?, ?, ?);

-- name: InsertManifestPolicyTemplateIcon :exec
INSERT INTO manifest_policy_template_icons (manifest_dir_name, policy_template_name, src, title, size, type, dark_mode)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: InsertManifestPolicyTemplateInput :exec
INSERT INTO manifest_policy_template_inputs (manifest_dir_name, policy_template_name, type, title, description, input_group, template_path, multi)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertManifestPolicyTemplateInputVar :exec
INSERT INTO manifest_policy_template_input_vars (manifest_dir_name, policy_template_name, input_title, name, default_value, description, type, title, multi, required, secret, show_user)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertBuildManifest :exec
INSERT INTO build_manifests (manifest_dir_name, ecs_reference, ecs_import_mappings)
VALUES (?, ?, ?);

-- name: InsertDataStream :exec
INSERT INTO data_streams (manifest_dir_name, data_stream_name, dataset, dataset_is_prefix, ilm_policy, release, title, type)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertStream :exec
INSERT INTO streams (manifest_dir_name, data_stream_name, title, input, description, template_path, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: InsertIngestPipeline :exec
INSERT INTO ingest_pipelines (manifest_dir_name, data_stream_name, pipeline_name, description, version)
VALUES (?, ?, ?, ?, ?);

-- name: InsertProcessor :exec
INSERT INTO processors (manifest_dir_name, data_stream_name, pipeline_name, processor_order, type, attributes, on_failure)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: InsertChangelog :exec
INSERT INTO changelog (manifest_dir_name, version, description, type, link)
VALUES (?, ?, ?, ?, ?);
