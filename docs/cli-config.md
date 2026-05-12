# CLI configuration

The reference CLI reads one shared configuration file and merges it with environment variables and command-line flags before constructing a single `mlx-go-sdk` client.

This document describes:

- where the CLI looks for config
- how precedence works
- which environment variables are supported
- the JSON config schema
- built-in defaults
- the explicit authentication rule

## Authentication rule

Authentication is intentionally simple:

- the CLI reads `MLX_TOKEN` from the environment
- the config file does **not** accept a token field
- there is no interactive login flow
- there is no `--token` flag

If `MLX_TOKEN` is missing, commands that need an SDK client fail fast.

## Config file location

The CLI resolves the config path in this order:

1. `--config <path>`
2. `MLX_CONFIG_FILE`
3. the default user config location

Default file name:

- `config.json`

Default config directory:

- `<user-config-dir>/mlx-go-sdk/`

Typical resolved paths are:

- Windows: `%AppData%\mlx-go-sdk\config.json`
- macOS: `~/Library/Application Support/mlx-go-sdk/config.json`
- Linux: `${XDG_CONFIG_HOME}/mlx-go-sdk/config.json`
- Linux fallback: `~/.config/mlx-go-sdk/config.json`

Useful commands:

- `mlx config path` — print the resolved path
- `mlx config init` — write a default config file
- `mlx config show` — print the effective config after file + environment overrides

## Precedence

The CLI uses the following precedence model:

1. command-line flags
2. environment variables
3. config file
4. built-in defaults

Authentication is the one exception:

- `MLX_TOKEN` comes only from the environment

## Supported environment variables

### Authentication

- `MLX_TOKEN` — required auth token

### Endpoint overrides

These match the SDK environment variables and override the config file when set:

- `MLX_BASE_URL`
- `MLX_LAUNCHER_URL`
- `MLX_COOKIES_URL`
- `MLX_PROXY_URL`

### CLI-specific overrides

- `MLX_CONFIG_FILE` — config file path override
- `MLX_OUTPUT` — output format override
- `MLX_TIMEOUT` — shared HTTP timeout override
- `MLX_USER_AGENT` — user agent override

## File format

The CLI currently accepts a single JSON document.

Important behavior:

- unknown fields are rejected
- multiple JSON values in one file are rejected
- duration values may be:
  - a Go duration string such as `30s`, `500ms`, or `2m`
  - a number, interpreted as seconds

## Example config

    {
      "version": "1",
      "endpoints": {
        "base_url": "https://api.multilogin.com/",
        "launcher_url": "https://launcher.mlx.yt:45001/",
        "cookies_url": "https://cookies.multilogin.com/",
        "proxy_url": "https://profile-proxy.multilogin.com/"
      },
      "transport": {
        "timeout": "30s",
        "user_agent": "mlx-go-sdk-cli"
      },
      "retry": {
        "enabled": true,
        "max_attempts": 4,
        "initial_interval": "500ms",
        "max_interval": "3s",
        "multiplier": 2,
        "jitter": 0.2
      },
      "poll": {
        "initial_interval": "2s",
        "max_interval": "10s",
        "timeout": "2m",
        "multiplier": 1.5
      },
      "output": {
        "format": "table",
        "pretty": true,
        "color": "auto"
      },
      "defaults": {
        "folder": {
          "id": "",
          "name": "Default folder"
        },
        "profile": {
          "browser_type": "mimic",
          "os_type": "windows",
          "storage_type": "all"
        },
        "launcher": {
          "automation_type": "playwright",
          "headless": false,
          "strict_mode": false,
          "wait_for_running": false
        },
        "export": {
          "root_dir": "",
          "stop_before_export": true,
          "ignore_stop_not_ready": false
        },
        "import": {
          "is_local": false,
          "wait": false
        },
        "extension": {
          "browser_type": "mimic",
          "storage_type": "cloud",
          "require_profile_usage_read": false
        },
        "cookies": {
          "target_website": "",
          "additional_website": "",
          "create_metadata_if_missing": true,
          "import_advanced_cookies": false,
          "strict_mode": false
        },
        "proxy": {
          "protocol": "socks5",
          "session_type": "sticky",
          "country": "",
          "region": "",
          "city": "",
          "prefer_socks5": true,
          "save_traffic": false,
          "patch_profile": true
        }
      }
    }

## Schema

## `version`

Type: `string`

Current schema version: `1`

The CLI normalizes missing `version` to the current schema version.

## `endpoints`

Optional URL overrides for the shared SDK client.

Fields:

- `base_url`
- `launcher_url`
- `cookies_url`
- `proxy_url`

Rules:

- each value must include scheme and host when set
- empty values mean “use the SDK default unless an environment override is present”

Examples:

- `https://api.multilogin.com/`
- `https://launcher.mlx.yt:45001/`

## `transport`

Shared transport settings.

Fields:

- `timeout`
- `user_agent`

### `transport.timeout`

Type: duration

Examples:

- `"30s"`
- `"2m"`
- `30`

Rules:

- must be greater than zero

### `transport.user_agent`

Type: `string`

Rules:

- must not be empty after normalization

## `retry`

Shared retry policy passed into the SDK client.

Fields:

- `enabled`
- `max_attempts`
- `initial_interval`
- `max_interval`
- `multiplier`
- `jitter`

### `retry.enabled`

Type: `boolean`

When `true`, the CLI enables SDK retry behavior with the configured settings.

### `retry.max_attempts`

Type: `integer`

Rules:

- must be greater than zero when retries are enabled

### `retry.initial_interval`

Type: duration

Rules:

- must be greater than zero when retries are enabled

### `retry.max_interval`

Type: duration

Rules:

- must be greater than or equal to `retry.initial_interval`

### `retry.multiplier`

Type: `number`

Rules:

- must be greater than `1`

### `retry.jitter`

Type: `number`

Rules:

- must not be negative

## `poll`

Defaults for workflow polling helpers.

Fields:

- `initial_interval`
- `max_interval`
- `timeout`
- `multiplier`

Rules:

- all durations must be greater than zero
- `max_interval` must be greater than or equal to `initial_interval`
- `multiplier` must be greater than `1`

These values are used by commands that wait for state transitions, such as:

- launcher start with `--wait`
- import with `--wait`
- workflow-backed export and extension commands

## `output`

Default renderer behavior.

Fields:

- `format`
- `pretty`
- `color`

### `output.format`

Supported values:

- `table`
- `json`
- `yaml`

Notes:

- `table` is the default human-oriented format
- `json` is the most stable choice for scripting
- `yaml` affects command output only; the config file itself is still JSON

### `output.pretty`

Type: `boolean`

Controls pretty JSON output when `format=json`.

### `output.color`

Supported values:

- `auto`
- `always`
- `never`

The current scaffold validates this field and preserves it in the effective config even where a command does not yet use terminal color styling.

## `defaults`

Per-domain command defaults.

## `defaults.folder`

Fields:

- `id`
- `name`

Typical use:

- default folder lookup context for name-based profile operations
- default human folder name in examples and config templates

## `defaults.profile`

Fields:

- `browser_type`
- `os_type`
- `storage_type`

### `defaults.profile.storage_type`

Supported values:

- `all`
- `local`
- `cloud`

Used as the default storage scope for profile lookups and list commands.

## `defaults.launcher`

Fields:

- `automation_type`
- `headless`
- `strict_mode`
- `wait_for_running`

### `defaults.launcher.automation_type`

Supported values:

- `selenium`
- `playwright`
- `puppeteer`
- `rod`

Default: `playwright`

This matches the reference CLI’s production-oriented default.

## `defaults.export`

Fields:

- `root_dir`
- `stop_before_export`
- `ignore_stop_not_ready`

Notes:

- `root_dir` is a good place to set a standard export destination for your environment
- `stop_before_export` defaults to `true`
- `ignore_stop_not_ready` controls whether a stop-before-export step tolerates a not-running profile

## `defaults.import`

Fields:

- `is_local`
- `wait`

Notes:

- `is_local` controls the default import storage mode
- `wait` controls whether import commands verify the imported profile by default

## `defaults.extension`

Fields:

- `browser_type`
- `storage_type`
- `require_profile_usage_read`

### `defaults.extension.storage_type`

Supported values:

- `local`
- `cloud`

Default: `cloud`

This follows the project’s live-validation guidance: cloud-backed extension objects are the safest default reference for profile attachment workflows.

## `defaults.cookies`

Fields:

- `target_website`
- `additional_website`
- `create_metadata_if_missing`
- `import_advanced_cookies`
- `strict_mode`

These values drive the high-level cookie seed/import workflows.

## `defaults.proxy`

Fields:

- `protocol`
- `session_type`
- `country`
- `region`
- `city`
- `prefer_socks5`
- `save_traffic`
- `patch_profile`

### `defaults.proxy.protocol`

Supported values:

- `socks5`
- `http`

Default: `socks5`

### `defaults.proxy.session_type`

Supported values:

- `sticky`
- `rotating`

Default: `sticky`

Notes:

- `prefer_socks5=true` is useful when you want the CLI to bias toward MLX SOCKS5 proxy generation
- `patch_profile=true` is the default for proxy assignment flows

## Built-in defaults

If a field is omitted from the config file and not overridden elsewhere, the CLI uses the following built-in defaults.

### Endpoints

- SDK defaults are used unless overridden

### Transport

- `timeout = 30s`
- `user_agent = mlx-go-sdk-cli`

### Retry

- `enabled = true`
- `max_attempts = 4`
- `initial_interval = 500ms`
- `max_interval = 3s`
- `multiplier = 2`
- `jitter = 0.2`

### Poll

- `initial_interval = 2s`
- `max_interval = 10s`
- `timeout = 2m`
- `multiplier = 1.5`

### Output

- `format = table`
- `pretty = true`
- `color = auto`

### Folder defaults

- `name = Default folder`

### Profile defaults

- `browser_type = mimic`
- `os_type = windows`
- `storage_type = all`

### Launcher defaults

- `automation_type = playwright`
- `headless = false`
- `strict_mode = false`
- `wait_for_running = false`

### Export defaults

- `stop_before_export = true`
- `ignore_stop_not_ready = false`

### Import defaults

- `is_local = false`
- `wait = false`

### Extension defaults

- `browser_type = mimic`
- `storage_type = cloud`
- `require_profile_usage_read = false`

### Cookies defaults

- `create_metadata_if_missing = true`
- `import_advanced_cookies = false`
- `strict_mode = false`

### Proxy defaults

- `protocol = socks5`
- `session_type = sticky`
- `prefer_socks5 = true`
- `save_traffic = false`
- `patch_profile = true`

## Validation behavior

The CLI validates the resolved config after merging file content and environment overrides.

Validation includes:

- URL shape checks for configured endpoints
- positive duration checks
- retry and poll interval consistency
- supported enum values for output, storage, automation, and proxy settings

The loader also rejects:

- unknown JSON fields
- malformed JSON
- multiple top-level JSON documents

## Practical examples

## Minimal config

    {
      "transport": {
        "user_agent": "acme-mlx-cli/1.0"
      },
      "output": {
        "format": "json"
      }
    }

This relies on:

- `MLX_TOKEN` from the environment
- SDK default endpoints
- built-in retry and poll defaults

## Team config with launcher/export defaults

    {
      "transport": {
        "timeout": "45s",
        "user_agent": "team-ops-cli/2.1"
      },
      "output": {
        "format": "table"
      },
      "defaults": {
        "folder": {
          "id": "folder-123"
        },
        "launcher": {
          "automation_type": "playwright",
          "headless": true,
          "strict_mode": false,
          "wait_for_running": true
        },
        "export": {
          "root_dir": "C:\\mlx-exports",
          "stop_before_export": true,
          "ignore_stop_not_ready": true
        }
      }
    }

## Local profile workflow defaults

    {
      "defaults": {
        "profile": {
          "storage_type": "local"
        },
        "import": {
          "is_local": true,
          "wait": true
        },
        "proxy": {
          "protocol": "socks5",
          "session_type": "sticky",
          "country": "us",
          "prefer_socks5": true,
          "patch_profile": true
        }
      }
    }

## Recommended operational guidance

- Keep `MLX_TOKEN` outside the config file.
- Prefer `json` output for automation and scripting.
- Set endpoint overrides only when you actually need non-default MLX endpoints.
- Keep retry enabled for normal operator use unless you have a very specific reason to disable it.
- Use explicit defaults for `storage_type`, launcher automation, export root directory, and proxy session mode so your CLI workflows stay predictable.
- When your workflow depends on confirmed state, prefer commands that wait and verify rather than assuming success from request acceptance alone.

## Related docs

- `docs/cli-reference.md`
- `docs/consumer-guide.md`
- `docs/verified-workflows.md`
- `docs/extensions.md`
- `docs/proxy-workflows.md`
- `docs/retries.md`
