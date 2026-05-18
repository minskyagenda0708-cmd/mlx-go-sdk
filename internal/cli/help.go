package cli

import (
	"fmt"
	"io"
)

func printHelpForCommand(name string, w io.Writer) error {
	switch name {
	case "config":
		printConfigHelp(w)
	case "folder":
		printFolderHelp(w)
	case "template":
		printTemplateHelp(w)
	case "launcher":
		printLauncherHelp(w)
	case "profile":
		printProfileHelp(w)
	case "export":
		printExportHelp(w)
	case "import":
		printImportHelp(w)
	case "extension":
		printExtensionHelp(w)
	case "cookies":
		printCookiesHelp(w)
	case "proxy":
		printProxyHelp(w)
	default:
		return fmt.Errorf("unknown command %q", name)
	}

	return nil
}

func printRootHelp(w io.Writer) {
	fmt.Fprint(w, `Reference CLI for mlx-go-sdk

Usage:
  mlx [global flags] <command> <subcommand> [flags]

Global flags:
  --config <path>   Path to CLI config file
  --output <fmt>    Output format override: table, json, yaml
  -h, --help        Show help
  --version         Show version

Commands:
  config       Config inspection and initialization
  folder       Folder CRUD operations
  template     Profile template discovery and inspection
  launcher     Launcher health and profile runtime control
  profile      Profile CRUD, lookup, and summary commands
  export       Export workflows and job status
  import       Import workflows and job status
  extension    Extension resource workflows
  cookies      Cookie metadata, import/export, and seeding
  proxy        Proxy usage, generation, and assignment
  version      Print CLI version
  help         Show help for a command

Examples:
  mlx config show
  mlx folder list
  mlx template list
  mlx template get --id tpl-123
  mlx launcher health
  mlx profile list --search Demo
  mlx profile create --template-id tpl-123 --name "Demo Local" --local --managed-proxy --proxy-country us
  mlx export run --profile-name Demo --root-dir C:\exports
  mlx import run --import-path C:\exports\demo.zip --wait
  mlx extension enable --id ext-1 --profile-name Demo
  mlx cookies seed --profile-name Demo --target-website google
  mlx proxy assign --profile-name Demo --country us --region new_jersey
`)
}

func printConfigHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx config <subcommand> [flags]

Subcommands:
  path                Print the resolved config path
  show                Print the effective config
  init                Write a default config file
`)
}

func printFolderHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx folder <subcommand> [flags]

Subcommands:
  list
  create --name <name> [--comment <text>]
  update --id <folder-id> --name <name> [--comment <text>]
  delete --ids <id1,id2,...>
`)
}

func printTemplateHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx template <subcommand> [flags]

Subcommands:
  list [--name <text>] [--limit <n>] [--offset <n>] [--trashbin]
  get --id <template-id>
`)
}

func printLauncherHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx launcher <subcommand> [flags]

Subcommands:
  health
  version
  status --profile-id <id> | --profile-name <name>
  statuses
  start --profile-id <id> | --profile-name <name> [--folder-id <id>] [--automation-type <type>] [--headless] [--strict] [--wait]
  stop --profile-id <id> | --profile-name <name> [--folder-id <id>] [--ignore-already-stopped] [--wait]
  stop-all [--type <cloud|local|quick>]
`)
}

func printProfileHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx profile <subcommand> [flags]

Subcommands:
  list [--search <text>] [--removed] [--limit <n>] [--offset <n>] [--storage-type <all|local|cloud>]
  get --id <id> | --name <name>
  create --file <request.json> [--wait]
  create --template-id <template-id> --name <name> [--folder-id <id>] [--local] [--managed-proxy] [--proxy-country <code>] [--proxy-region <name>] [--proxy-city <name>]
  update --file <request.json>
  patch --file <request.json>
  clone --id <id> | --name <name> [--times <n>]
  move --ids <id1,id2,...> --dest-folder-id <folder-id>
  delete --ids <id1,id2,...> [--permanently]
  restore --ids <id1,id2,...>
  summary --id <id> | --name <name>
`)
}

func printExportHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx export <subcommand> [flags]

Subcommands:
  run --root-dir <dir> (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--folder-name <name>] [--profile-name-override <name>] [--stop-before-export] [--ignore-stop-not-ready]
  status --export-id <id>
  statuses
`)
}

func printImportHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx import <subcommand> [flags]

Subcommands:
  run --import-path <archive.zip> [--is-local] [--wait]
  status --import-id <id>
  statuses
`)
}

func printExtensionHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx extension <subcommand> [flags]

Subcommands:
  list [--name <text>] [--limit <n>] [--offset <n>] [--trashbin]
  get --id <resource-id>
  upload --path <zip> [--storage-type <cloud|local>]
  create-url --url <download-url> [--browser-type <browser>] [--storage-type <cloud|local>]
  create-webstore --extension-id <id> [--browser-type <browser>] [--storage-type <cloud|local>]
  enable --id <resource-id> (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--require-profile-usage-read]
  disable --id <resource-id> (--profile-id <id> | --profile-name <name>) [--folder-id <id>]
  usages --id <resource-id> | (--profile-id <id> | --profile-name <name>)
  download --id <resource-id>
  delete --id <resource-id> [--permanently]
  restore --id <resource-id>
`)
}

func printCookiesHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx cookies <subcommand> [flags]

Subcommands:
  websites
  list --profile-id <id> | --profile-name <name>
  metadata create (--profile-id <id> | --profile-name <name>) --target-website <key> [--strict]
  metadata update (--profile-id <id> | --profile-name <name>) --target-website <key> [--additional-website <key>] [--strict]
  import (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--advanced] [--strict] [--cookies-file <cookies.json>]
  export (--profile-id <id> | --profile-name <name>) [--folder-id <id>]
  seed (--profile-id <id> | --profile-name <name>) [--folder-id <id>] --target-website <key> [--additional-website <key>] [--create-metadata-if-missing] [--advanced] [--strict]
`)
}

func printProxyHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  mlx proxy <subcommand> [flags]

Subcommands:
  usage
  generate [--country <code>] [--region <name>] [--city <name>] [--protocol <socks5|http>] [--session-type <sticky|rotating>] [--ip-ttl <seconds>] [--count <n>] [--strict]
  assign (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--country <code>] [--region <name>] [--city <name>] [--protocol <socks5|http>] [--session-type <sticky|rotating>] [--ip-ttl <seconds>] [--strict] [--prefer-socks5] [--save-traffic] [--patch-profile]
`)
}
