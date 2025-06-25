CREATE TABLE IF NOT EXISTS manifests (
	dir_name TEXT PRIMARY KEY, -- The directory name of the package.
	name TEXT, -- The name of the package.
	title TEXT, -- Title of the package.
	version TEXT, -- The version of the package.
	release TEXT, -- The release stage of the package (e.g. experimental, ga).
	description TEXT, -- A long description of the package.
	type TEXT, -- The type of package (e.g. integration, input).
	format_version TEXT, -- The format version of the manifest.
	license TEXT, -- The license of the package.
	kibana_version TEXT, -- Kibana versions compatible with this package.
	elastic_subscription TEXT, -- The subscription required for this package (e.g. basic, gold, platinum, enterprise).
	source_license TEXT,
	owner_github TEXT, -- The GitHub ID of the owner of the package.
	owner_type TEXT -- The type of owner (e.g. elastic).
);

CREATE TABLE IF NOT EXISTS manifest_icons (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	src TEXT, -- The path to the icon file.
	title TEXT, -- The title of the icon.
	size TEXT, -- The size of the icon.
	type TEXT, -- The media type of the icon.
	dark_mode BOOLEAN, -- Whether the icon is for dark mode.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_categories (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	category TEXT, -- Category to which this package belongs.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_screenshots (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	src TEXT, -- The path to the screenshot file.
	title TEXT, -- The title of the screenshot.
	size TEXT, -- The size of the screenshot.
	type TEXT, -- The media type of the screenshot.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_vars (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	name TEXT, -- The name of the variable.
	default_value TEXT, -- The default value of the variable.
	description TEXT, -- A description of the variable.
	type TEXT, -- The type of the variable.
	title TEXT, -- The title of the variable.
	multi BOOLEAN, -- Whether the variable can have multiple values.
	required BOOLEAN, -- Whether the variable is required.
	secret BOOLEAN, -- Whether the variable is a secret.
	show_user BOOLEAN, -- Whether the variable should be shown to the user.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_policy_templates (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	name TEXT, -- The name of the policy template.
	title TEXT, -- The title of the policy template.
	description TEXT, -- A description of the policy template.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_var_options (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	var_name TEXT, -- The name of the variable this option belongs to.
	value TEXT, -- The value of the option.
	text TEXT, -- The text to display for the option.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_policy_template_categories (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	policy_template_name TEXT, -- The name of the policy template this category belongs to.
	category TEXT, -- Category to which this policy template belongs.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_policy_template_data_streams (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	policy_template_name TEXT, -- The name of the policy template this data stream belongs to.
	data_stream TEXT, -- Data streams associated with this policy template.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_policy_template_icons (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	policy_template_name TEXT, -- The name of the policy template this icon belongs to.
	src TEXT, -- The path to the icon file.
	title TEXT, -- The title of the icon.
	size TEXT, -- The size of the icon.
	type TEXT, -- The media type of the icon.
	dark_mode BOOLEAN, -- Whether the icon is for dark mode.
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_policy_template_inputs (
	manifest_dir_name TEXT, -- Foreign key to manifests table.
	policy_template_name TEXT, -- The name of the policy template this input belongs to.
	type TEXT, -- The type of the input.
	title TEXT, -- The title of the input.
	description TEXT, -- A description of the input.
	input_group TEXT, -- The input group this input belongs to.
	template_path TEXT, -- The path to the input template.
	multi BOOLEAN,
	FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS manifest_policy_template_input_vars (
    manifest_dir_name TEXT, -- Foreign key to manifests table.
    policy_template_name TEXT, -- The name of the policy template this input var belongs to.
    input_title TEXT, -- The title of the input this var belongs to.
    name TEXT, -- The name of the variable.
	default_value TEXT, -- The default value of the variable.
	description TEXT, -- A description of the variable.
	type TEXT, -- The type of the variable.
	title TEXT, -- The title of the variable.
	multi BOOLEAN, -- Whether the variable can have multiple values.
	required BOOLEAN, -- Whether the variable is required.
	secret BOOLEAN, -- Whether the variable is a secret.
	show_user BOOLEAN, -- Whether the variable should be shown to the user.
    FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS build_manifests (
    manifest_dir_name TEXT PRIMARY KEY, -- Foreign key to manifests table.
    ecs_reference TEXT, -- Source reference for the ECS dependency (e.g. git@github.com:elastic/ecs.git#v1.12.0).
    ecs_import_mappings BOOLEAN, -- Whether or not to import common used dynamic templates and properties into the package.
    FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS data_streams (
    manifest_dir_name TEXT NOT NULL, -- Foreign key to manifests table.
    data_stream_name TEXT NOT NULL, -- The name of the data stream directory.
    dataset TEXT,
    dataset_is_prefix BOOLEAN,
    ilm_policy TEXT,
    release TEXT,
    title TEXT,
    type TEXT,
    PRIMARY KEY (manifest_dir_name, data_stream_name),
    FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);

CREATE TABLE IF NOT EXISTS streams (
    manifest_dir_name TEXT NOT NULL, -- Foreign key to manifests table.
    data_stream_name TEXT NOT NULL, -- Foreign key to data_streams table.
    title TEXT,
    input TEXT,
    description TEXT,
    template_path TEXT,
    enabled BOOLEAN,
    FOREIGN KEY(manifest_dir_name, data_stream_name) REFERENCES data_streams(manifest_dir_name, data_stream_name)
);

CREATE TABLE IF NOT EXISTS ingest_pipelines (
    manifest_dir_name TEXT NOT NULL, -- Foreign key to manifests table.
    data_stream_name TEXT NOT NULL, -- Foreign key to data_streams table.
    pipeline_name TEXT NOT NULL, -- The name of the pipeline file (e.g. default.yml).
    description TEXT,
    version INTEGER,
    PRIMARY KEY (manifest_dir_name, data_stream_name, pipeline_name),
    FOREIGN KEY(manifest_dir_name, data_stream_name) REFERENCES data_streams(manifest_dir_name, data_stream_name)
);

CREATE TABLE IF NOT EXISTS processors (
    manifest_dir_name TEXT NOT NULL, -- Foreign key to manifests table.
    data_stream_name TEXT NOT NULL, -- Foreign key to data_streams table.
    pipeline_name TEXT, -- Foreign key to ingest_pipelines table.
    processor_order INTEGER, -- The order of the processor in the pipeline.
    type TEXT, -- The type of the processor.
    attributes JSON, -- The attributes of the processor.
    on_failure BOOLEAN, -- Whether this is an on_failure processor.
    FOREIGN KEY(manifest_dir_name, data_stream_name, pipeline_name) REFERENCES ingest_pipelines(manifest_dir_name, data_stream_name, pipeline_name)
);

CREATE TABLE IF NOT EXISTS changelog (
    manifest_dir_name TEXT NOT NULL, -- Foreign key to manifests table.
    version TEXT,
    description TEXT,
    type TEXT,
    link TEXT,
    FOREIGN KEY(manifest_dir_name) REFERENCES manifests(dir_name)
);
