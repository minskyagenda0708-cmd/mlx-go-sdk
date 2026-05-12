package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	mlx "mlx-go-sdk"
)

const CLIVersion = "dev"

type globalOptions struct {
	ConfigPath string
	Output     string
	Help       bool
	Version    bool
}

type resolvedProfile struct {
	ID       string
	Name     string
	FolderID string
	Profile  *mlx.Profile
	Meta     *mlx.ProfileMeta
}

func Main() {
	if err := Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func Execute(args []string) error {
	global, rest, err := parseGlobal(args)
	if err != nil {
		return err
	}

	if global.Version && len(rest) == 0 {
		fmt.Fprintln(os.Stdout, CLIVersion)
		return nil
	}

	if global.Help && len(rest) == 0 {
		printRootHelp(os.Stdout)
		return nil
	}

	if len(rest) == 0 {
		printRootHelp(os.Stdout)
		return nil
	}

	cmd := rest[0]
	subArgs := rest[1:]

	switch cmd {
	case "help":
		if len(subArgs) == 0 {
			printRootHelp(os.Stdout)
			return nil
		}
		return printHelpForCommand(subArgs[0], os.Stdout)
	case "version":
		fmt.Fprintln(os.Stdout, CLIVersion)
		return nil
	case "config":
		return runConfig(subArgs, global)
	case "folder":
		return runFolder(subArgs, global)
	case "template":
		return runTemplate(subArgs, global)
	case "launcher":
		return runLauncher(subArgs, global)
	case "profile":
		return runProfile(subArgs, global)
	case "export":
		return runExport(subArgs, global)
	case "import":
		return runImport(subArgs, global)
	case "extension":
		return runExtension(subArgs, global)
	case "cookies":
		return runCookies(subArgs, global)
	case "proxy":
		return runProxy(subArgs, global)
	default:
		printRootHelp(os.Stdout)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func parseGlobal(args []string) (globalOptions, []string, error) {
	var opts globalOptions

	fs := flag.NewFlagSet("mlx", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.ConfigPath, "config", "", "path to CLI config file")
	fs.StringVar(&opts.Output, "output", "", "output format override: table, json, yaml")
	fs.BoolVar(&opts.Help, "help", false, "show help")
	fs.BoolVar(&opts.Help, "h", false, "show help")
	fs.BoolVar(&opts.Version, "version", false, "show version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return opts, nil, nil
		}
		return opts, nil, err
	}

	rest, err := extractTrailingGlobalFlags(fs.Args(), &opts)
	if err != nil {
		return opts, nil, err
	}

	return opts, rest, nil
}

func extractTrailingGlobalFlags(args []string, opts *globalOptions) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if opts == nil {
		opts = &globalOptions{}
	}

	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return nil, errors.New("flag needs an argument: -config")
			}
			opts.ConfigPath = args[i+1]
			i++
		case strings.HasPrefix(arg, "--config="):
			opts.ConfigPath = strings.TrimPrefix(arg, "--config=")
		case arg == "--output":
			if i+1 >= len(args) {
				return nil, errors.New("flag needs an argument: -output")
			}
			opts.Output = args[i+1]
			i++
		case strings.HasPrefix(arg, "--output="):
			opts.Output = strings.TrimPrefix(arg, "--output=")
		default:
			rest = append(rest, arg)
		}
	}

	return rest, nil
}

func withRuntime(global globalOptions, fn func(*Runtime) error) error {
	rt, err := LoadRuntime(global.ConfigPath)
	if err != nil {
		return err
	}
	if override := strings.ToLower(strings.TrimSpace(global.Output)); override != "" {
		rt.Config.Output.Format = override
		rt.Config = rt.Config.Normalize()
		if err := rt.Config.Validate(); err != nil {
			return err
		}
	}
	return fn(rt)
}

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

func runConfig(args []string, global globalOptions) error {
	if len(args) == 0 {
		printConfigHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "path":
		return runConfigPath(args[1:], global)
	case "show":
		return runConfigShow(args[1:], global)
	case "init":
		return runConfigInit(args[1:], global)
	default:
		printConfigHelp(os.Stdout)
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func runConfigPath(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("config path", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx config path")
		return nil
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	path, err := ResolveConfigPath(global.ConfigPath)
	if err != nil {
		return err
	}
	return emitWithGlobal(global, map[string]any{"path": path})
}

func runConfigShow(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx config show")
		return nil
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	cfg, err := LoadConfig(global.ConfigPath)
	if err != nil {
		return err
	}
	return emitWithGlobal(global, cfg)
}

func runConfigInit(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", "", "path to write")
	force := fs.Bool("force", false, "overwrite existing file")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx config init [--path <path>] [--force]")
		return nil
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	path, err := ResolveConfigPath(firstNonEmpty(*pathFlag, global.ConfigPath))
	if err != nil {
		return err
	}
	if !*force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists: %s", path)
		}
	}

	body, err := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return emitWithGlobal(global, map[string]any{
		"path":    path,
		"written": true,
	})
}

func runFolder(args []string, global globalOptions) error {
	if len(args) == 0 {
		printFolderHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return runFolderList(args[1:], global)
	case "create":
		return runFolderCreate(args[1:], global)
	case "update":
		return runFolderUpdate(args[1:], global)
	case "delete":
		return runFolderDelete(args[1:], global)
	default:
		printFolderHelp(os.Stdout)
		return fmt.Errorf("unknown folder subcommand %q", args[0])
	}
}

func runFolderList(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("folder list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx folder list")
		return nil
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Folders.List(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp.Data.Folders)
	})
}

func runFolderCreate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("folder create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "folder name")
	comment := fs.String("comment", "", "folder comment")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx folder create --name <name> [--comment <text>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("--name is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Folders.Create(context.Background(), &mlx.CreateFolderRequest{
			Name:    *name,
			Comment: *comment,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runFolderUpdate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("folder update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "folder id")
	name := fs.String("name", "", "folder name")
	comment := fs.String("comment", "", "folder comment")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx folder update --id <folder-id> --name <name> [--comment <text>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("--name is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Folders.Update(context.Background(), &mlx.UpdateFolderRequest{
			FolderID: *id,
			Name:     *name,
			Comment:  *comment,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runFolderDelete(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("folder delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ids := fs.String("ids", "", "comma-separated folder ids")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx folder delete --ids <id1,id2,...>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	idList := splitCSV(*ids)
	if len(idList) == 0 {
		return errors.New("--ids is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Folders.Delete(context.Background(), &mlx.DeleteFoldersRequest{IDs: idList})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runTemplate(args []string, global globalOptions) error {
	if len(args) == 0 {
		printTemplateHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return runTemplateList(args[1:], global)
	case "get":
		return runTemplateGet(args[1:], global)
	default:
		printTemplateHelp(os.Stdout)
		return fmt.Errorf("unknown template subcommand %q", args[0])
	}
}

func runTemplateList(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("template list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "template name filter")
	limit := fs.Int("limit", 50, "page size")
	offset := fs.Int("offset", 0, "page offset")
	trashbin := fs.Bool("trashbin", false, "show trashbin templates only")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx template list [--name <text>] [--limit <n>] [--offset <n>] [--trashbin]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		tb := *trashbin
		resp, _, err := rt.Client.Resources.ListProfileTemplates(context.Background(), &mlx.ListResourceMetasOptions{
			ObjectName: *name,
			Limit:      *limit,
			Offset:     *offset,
			Trashbin:   &tb,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp.Data.Objects)
	})
}

func runTemplateGet(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("template get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "template resource id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx template get --id <template-id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		metaResp, _, err := rt.Client.Resources.GetMeta(context.Background(), *id)
		if err != nil {
			return err
		}
		downloadResp, _, err := rt.Client.Resources.Download(context.Background(), *id)
		if err != nil {
			return err
		}
		templateDoc, err := loadProfileTemplate(metaResp.Data.MetaInfo, downloadResp.Path)
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{
			"meta":     metaResp.Data,
			"path":     downloadResp.Path,
			"template": templateDoc,
		})
	})
}

func runLauncher(args []string, global globalOptions) error {
	if len(args) == 0 {
		printLauncherHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "health":
		return runLauncherHealth(args[1:], global)
	case "version":
		return runLauncherVersion(args[1:], global)
	case "status":
		return runLauncherStatus(args[1:], global)
	case "statuses":
		return runLauncherStatuses(args[1:], global)
	case "start":
		return runLauncherStart(args[1:], global)
	case "stop":
		return runLauncherStop(args[1:], global)
	case "stop-all":
		return runLauncherStopAll(args[1:], global)
	default:
		printLauncherHelp(os.Stdout)
		return fmt.Errorf("unknown launcher subcommand %q", args[0])
	}
}

func runLauncherHealth(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher health", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher health")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Launcher.Health(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runLauncherVersion(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher version")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Launcher.Version(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runLauncherStatus(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher status --profile-id <id> | --profile-name <name>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Launcher.Status(context.Background(), profile.ID)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runLauncherStatuses(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher statuses", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher statuses")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Launcher.Statuses(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{
			"active_counter": resp.Data.ActiveCounter,
			"states":         resp.Data.States,
		})
	})
}

func runLauncherStart(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id")
	automationType := fs.String("automation-type", "", "selenium|playwright|puppeteer|rod")
	headless := newOptionalBoolFlag(fs, "headless", "start headless")
	strict := newOptionalBoolFlag(fs, "strict", "enable strict mode")
	wait := newOptionalBoolFlag(fs, "wait", "wait for running status")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher start --profile-id <id> | --profile-name <name> [--folder-id <id>] [--automation-type <type>] [--headless] [--strict] [--wait]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		startAutomation := strings.TrimSpace(*automationType)
		if startAutomation == "" {
			startAutomation = rt.Config.Defaults.Launcher.AutomationType
		}
		effectiveHeadless := headless.ValueOr(rt.Config.Defaults.Launcher.Headless)
		effectiveStrict := strict.ValueOr(rt.Config.Defaults.Launcher.StrictMode)
		effectiveWait := wait.ValueOr(rt.Config.Defaults.Launcher.WaitForRunning)
		opts := mlx.StartProfileOptions{
			AutomationType: mlx.AutomationType(startAutomation),
			Headless:       effectiveHeadless,
			StrictMode:     effectiveStrict,
		}

		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.StartProfileByName(context.Background(), *profileName, mlx.StartProfileByNameOptions{
				FindOptions:    buildFindOptions(rt.Config, *folderID),
				StartOptions:   opts,
				WaitForRunning: effectiveWait,
				PollOptions:    rt.Config.PollOptions(),
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		profile, err := resolveProfile(rt, *profileID, "", *folderID)
		if err != nil {
			return err
		}
		startResp, _, err := rt.Client.Launcher.Start(context.Background(), firstNonEmpty(*folderID, profile.FolderID), profile.ID, opts)
		if err != nil {
			return err
		}
		if !effectiveWait {
			return emit(rt, startResp)
		}
		statusResp, _, err := rt.Client.Launcher.WaitForRunning(context.Background(), profile.ID, rt.Config.PollOptions())
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{
			"profile":         profile,
			"start_response":  startResp,
			"runtime_status":  statusResp,
			"automation_type": startAutomation,
		})
	})
}

func runLauncherStop(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher stop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	ignoreAlreadyStopped := fs.Bool("ignore-already-stopped", false, "treat already-stopped errors as success")
	wait := fs.Bool("wait", false, "wait for non-running status")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher stop --profile-id <id> | --profile-name <name> [--folder-id <id>] [--ignore-already-stopped] [--wait]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.StopProfileByName(context.Background(), *profileName, mlx.StopProfileByNameOptions{
				FindOptions:          buildFindOptions(rt.Config, *folderID),
				IgnoreAlreadyStopped: *ignoreAlreadyStopped,
				WaitForStopped:       *wait,
				PollOptions:          rt.Config.PollOptions(),
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		profile, err := resolveProfile(rt, *profileID, "", *folderID)
		if err != nil {
			return err
		}
		stopResp, _, err := rt.Client.Launcher.Stop(context.Background(), profile.ID)
		if err != nil && !(*ignoreAlreadyStopped && isAlreadyStoppedError(err)) {
			return err
		}
		if !*wait {
			return emit(rt, map[string]any{
				"profile":       profile,
				"stop_response": stopResp,
			})
		}
		statusResp, err := waitForStoppedStatus(context.Background(), rt, profile.ID)
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{
			"profile":        profile,
			"stop_response":  stopResp,
			"runtime_status": statusResp,
		})
	})
}

func runLauncherStopAll(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher stop-all", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	kind := fs.String("type", "", "optional stop-all type filter")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx launcher stop-all [--type <cloud|local|quick>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Launcher.StopAll(context.Background(), mlx.StopAllProfilesOptions{Type: *kind})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfile(args []string, global globalOptions) error {
	if len(args) == 0 {
		printProfileHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return runProfileList(args[1:], global)
	case "get":
		return runProfileGet(args[1:], global)
	case "create":
		return runProfileCreate(args[1:], global)
	case "update":
		return runProfileUpdate(args[1:], global)
	case "patch":
		return runProfilePatch(args[1:], global)
	case "clone":
		return runProfileClone(args[1:], global)
	case "move":
		return runProfileMove(args[1:], global)
	case "delete":
		return runProfileDelete(args[1:], global)
	case "restore":
		return runProfileRestore(args[1:], global)
	case "summary":
		return runProfileSummary(args[1:], global)
	default:
		printProfileHelp(os.Stdout)
		return fmt.Errorf("unknown profile subcommand %q", args[0])
	}
}

func runProfileList(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	search := fs.String("search", "", "search text")
	removed := fs.Bool("removed", false, "search removed profiles")
	limit := fs.Int("limit", 100, "page size")
	offset := fs.Int("offset", 0, "page offset")
	storageType := fs.String("storage-type", "", "all|local|cloud")
	folderID := fs.String("folder-id", "", "folder id")
	browserType := fs.String("browser-type", "", "browser type")
	osType := fs.String("os-type", "", "os type")
	orderBy := fs.String("order-by", "", "order by field")
	sortOrder := fs.String("sort", "", "asc|desc")
	tags := fs.String("tags", "", "comma-separated tags")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile list [--search <text>] [--removed] [--limit <n>] [--offset <n>] [--storage-type <all|local|cloud>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		effectiveStorage := strings.TrimSpace(*storageType)
		if effectiveStorage == "" {
			effectiveStorage = rt.Config.Defaults.Profile.StorageType
		}
		resp, _, err := rt.Client.Profiles.Search(context.Background(), &mlx.SearchProfilesRequest{
			IsRemoved:   *removed,
			Limit:       *limit,
			Offset:      *offset,
			SearchText:  *search,
			StorageType: effectiveStorage,
			FolderID:    firstNonEmpty(*folderID, rt.Config.Defaults.Folder.ID),
			BrowserType: *browserType,
			OSType:      *osType,
			OrderBy:     *orderBy,
			Sort:        *sortOrder,
			Tags:        splitCSV(*tags),
		})
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{
			"total_count": resp.Data.TotalCount,
			"profiles":    resp.Data.Profiles,
		})
	})
}

func runProfileGet(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "profile id")
	name := fs.String("name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for name lookup")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile get --id <id> | --name <name>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*id, *name, "--id", "--name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *id, *name, *folderID)
		if err != nil {
			return err
		}
		return emit(rt, profile.Meta)
	})
}

func runProfileCreate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "path to CreateProfileRequest JSON")
	templateID := fs.String("template-id", "", "profile template resource id")
	name := fs.String("name", "", "profile name override for template-based creation")
	folderID := fs.String("folder-id", "", "folder id override for template-based creation")
	local := newOptionalBoolFlag(fs, "local", "create a local profile from the template")
	managedProxy := fs.Bool("managed-proxy", false, "generate and attach an MLX managed proxy during template-based creation")
	proxyCountry := fs.String("proxy-country", "", "proxy country code")
	proxyRegion := fs.String("proxy-region", "", "proxy region")
	proxyCity := fs.String("proxy-city", "", "proxy city")
	proxyProtocol := fs.String("proxy-protocol", "", "proxy protocol: socks5 or http")
	proxySessionType := fs.String("proxy-session-type", "", "proxy session type: sticky or rotating")
	proxyIPTTL := fs.Int("proxy-ip-ttl", 0, "proxy IPTTL")
	proxyStrict := fs.Bool("proxy-strict", false, "enable strict mode for managed proxy generation")
	proxySaveTraffic := newOptionalBoolFlag(fs, "proxy-save-traffic", "save traffic in the generated profile proxy")
	wait := fs.Bool("wait", false, "verify created profile metas")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile create --file <request.json> [--wait]\n       mlx profile create --template-id <template-id> --name <name> [--folder-id <id>] [--local] [--managed-proxy] [--proxy-country <code>] [--proxy-region <name>] [--proxy-city <name>] [--wait]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	if strings.TrimSpace(*file) != "" && strings.TrimSpace(*templateID) != "" {
		return errors.New("--file and --template-id are mutually exclusive")
	}
	if strings.TrimSpace(*file) == "" && strings.TrimSpace(*templateID) == "" {
		return errors.New("one of --file or --template-id is required")
	}

	var req mlx.CreateProfileRequest
	usingTemplate := strings.TrimSpace(*templateID) != ""

	return withRuntime(global, func(rt *Runtime) error {
		if !usingTemplate {
			if err := readJSONFile(*file, &req); err != nil {
				return err
			}
		} else {
			metaResp, _, err := rt.Client.Resources.GetMeta(context.Background(), *templateID)
			if err != nil {
				return err
			}
			downloadResp, _, err := rt.Client.Resources.Download(context.Background(), *templateID)
			if err != nil {
				return err
			}
			templateDoc, err := loadProfileTemplate(metaResp.Data.MetaInfo, downloadResp.Path)
			if err != nil {
				return err
			}

			resolvedFolderID, err := resolveFolderID(rt, *folderID)
			if err != nil {
				return err
			}
			templateReq, err := buildCreateProfileRequestFromTemplate(templateDoc, *name, resolvedFolderID, local.BoolPtr())
			if err != nil {
				return err
			}
			req = *templateReq

			if *managedProxy {
				generated, err := rt.Client.Proxies.GenerateProfileProxy(context.Background(), &mlx.GenerateProfileProxyRequest{
					GenerateProxyRequest: *buildGenerateProxyRequest(
						rt.Config,
						*proxyCountry,
						*proxyRegion,
						*proxyCity,
						*proxyProtocol,
						*proxySessionType,
						*proxyIPTTL,
						1,
						*proxyStrict,
					),
					PreferSOCKS5: strings.TrimSpace(*proxyProtocol) == "" && rt.Config.Defaults.Proxy.PreferSOCKS5,
					SaveTraffic:  proxySaveTraffic.ValueOr(rt.Config.Defaults.Proxy.SaveTraffic),
				})
				if err != nil {
					return err
				}
				if req.Parameters == nil {
					req.Parameters = &mlx.ProfileParameters{}
				}
				req.Parameters.Proxy = generated.ProfileProxy
			}
		}

		if *wait {
			resp, err := rt.Client.Workflows.CreateProfilesAndVerify(context.Background(), &req, mlx.CreateProfilesAndVerifyOptions{
				PollOptions: rt.Config.PollOptions(),
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}
		resp, _, err := rt.Client.Profiles.Create(context.Background(), &req)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfileUpdate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "path to UpdateProfileRequest JSON")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile update --file <request.json>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*file) == "" {
		return errors.New("--file is required")
	}

	var req mlx.UpdateProfileRequest
	if err := readJSONFile(*file, &req); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Profiles.Update(context.Background(), &req)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfilePatch(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile patch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "path to PatchProfileRequest JSON")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile patch --file <request.json>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*file) == "" {
		return errors.New("--file is required")
	}

	var req mlx.PatchProfileRequest
	if err := readJSONFile(*file, &req); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Profiles.Patch(context.Background(), &req)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfileClone(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile clone", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "profile id")
	name := fs.String("name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for name lookup")
	times := fs.Int("times", 1, "clone count")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile clone --id <id> | --name <name> [--times <n>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*id, *name, "--id", "--name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *id, *name, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Profiles.Clone(context.Background(), &mlx.CloneProfileRequest{
			ProfileID: profile.ID,
			Times:     *times,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfileMove(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile move", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ids := fs.String("ids", "", "comma-separated profile ids")
	destFolderID := fs.String("dest-folder-id", "", "destination folder id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile move --ids <id1,id2,...> --dest-folder-id <folder-id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	idList := splitCSV(*ids)
	if len(idList) == 0 {
		return errors.New("--ids is required")
	}
	if strings.TrimSpace(*destFolderID) == "" {
		return errors.New("--dest-folder-id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Profiles.Move(context.Background(), &mlx.MoveProfilesRequest{
			DestinationFolderID: *destFolderID,
			IDs:                 idList,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfileDelete(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ids := fs.String("ids", "", "comma-separated profile ids")
	permanently := fs.Bool("permanently", false, "permanent delete")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile delete --ids <id1,id2,...> [--permanently]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	idList := splitCSV(*ids)
	if len(idList) == 0 {
		return errors.New("--ids is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Profiles.Delete(context.Background(), &mlx.DeleteProfilesRequest{
			IDs:         idList,
			Permanently: *permanently,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfileRestore(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ids := fs.String("ids", "", "comma-separated profile ids")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile restore --ids <id1,id2,...>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	idList := splitCSV(*ids)
	if len(idList) == 0 {
		return errors.New("--ids is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Profiles.Restore(context.Background(), &mlx.RestoreProfilesRequest{IDs: idList})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProfileSummary(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("profile summary", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "profile id")
	name := fs.String("name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for name lookup")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx profile summary --id <id> | --name <name>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*id, *name, "--id", "--name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *id, *name, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Profiles.GetSummary(context.Background(), profile.ID)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runExport(args []string, global globalOptions) error {
	if len(args) == 0 {
		printExportHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "run":
		return runExportRun(args[1:], global)
	case "status":
		return runExportStatus(args[1:], global)
	case "statuses":
		return runExportStatuses(args[1:], global)
	default:
		printExportHelp(os.Stdout)
		return fmt.Errorf("unknown export subcommand %q", args[0])
	}
}

func runExportRun(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("export run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	rootDir := fs.String("root-dir", "", "export root dir")
	folderName := fs.String("folder-name", "", "archive folder name override")
	profileNameOverride := fs.String("profile-name-override", "", "archive profile name override")
	stopBeforeExport := newOptionalBoolFlag(fs, "stop-before-export", "stop profile before export")
	ignoreStopNotReady := newOptionalBoolFlag(fs, "ignore-stop-not-ready", "ignore stop errors for not-ready profiles")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx export run --root-dir <dir> (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--folder-name <name>] [--profile-name-override <name>] [--stop-before-export] [--ignore-stop-not-ready]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}
	if strings.TrimSpace(*rootDir) == "" {
		return errors.New("--root-dir is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		effectiveStopBeforeExport := stopBeforeExport.ValueOr(rt.Config.Defaults.Export.StopBeforeExport)
		effectiveIgnoreStopNotReady := ignoreStopNotReady.ValueOr(rt.Config.Defaults.Export.IgnoreStopNotReady)
		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.ExportProfileByNameToFolder(context.Background(), *profileName, mlx.ExportProfileByNameToFolderOptions{
				FindOptions: buildFindOptions(rt.Config, *folderID),
				ExportOptions: mlx.ExportProfileToFolderOptions{
					RootDir:      *rootDir,
					FolderName:   *folderName,
					ProfileName:  *profileNameOverride,
					PollInterval: rt.Config.Poll.InitialInterval.Duration(),
					WaitTimeout:  rt.Config.Poll.Timeout.Duration(),
				},
				StopBeforeExport:   effectiveStopBeforeExport,
				IgnoreStopNotReady: effectiveIgnoreStopNotReady,
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		profile, err := resolveProfile(rt, *profileID, "", *folderID)
		if err != nil {
			return err
		}
		if effectiveStopBeforeExport {
			if _, _, err := rt.Client.Launcher.Stop(context.Background(), profile.ID); err != nil && !(effectiveIgnoreStopNotReady && isAlreadyStoppedError(err)) {
				return err
			}
		}
		profileNameForArchive := firstNonEmpty(*profileNameOverride, profile.Name)
		resp, err := rt.Client.Archives.ExportProfileToFolder(context.Background(), profile.ID, mlx.ExportProfileToFolderOptions{
			RootDir:      *rootDir,
			FolderName:   *folderName,
			ProfileName:  profileNameForArchive,
			PollInterval: rt.Config.Poll.InitialInterval.Duration(),
			WaitTimeout:  rt.Config.Poll.Timeout.Duration(),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runExportStatus(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("export status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	exportID := fs.String("export-id", "", "export id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx export status --export-id <id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*exportID) == "" {
		return errors.New("--export-id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Transfers.ExportStatus(context.Background(), *exportID)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runExportStatuses(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("export statuses", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx export statuses")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Transfers.ExportStatuses(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp.Data.Statuses)
	})
}

func runImport(args []string, global globalOptions) error {
	if len(args) == 0 {
		printImportHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "run":
		return runImportRun(args[1:], global)
	case "status":
		return runImportStatus(args[1:], global)
	case "statuses":
		return runImportStatuses(args[1:], global)
	default:
		printImportHelp(os.Stdout)
		return fmt.Errorf("unknown import subcommand %q", args[0])
	}
}

func runImportRun(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("import run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	importPath := fs.String("import-path", "", "path to exported archive")
	isLocal := newOptionalBoolFlag(fs, "is-local", "import as local profile")
	wait := newOptionalBoolFlag(fs, "wait", "verify imported profile meta")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx import run --import-path <archive.zip> [--is-local] [--wait]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*importPath) == "" {
		return errors.New("--import-path is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		req := &mlx.ImportProfileRequest{
			ImportPath: *importPath,
			IsLocal:    isLocal.ValueOr(rt.Config.Defaults.Import.IsLocal),
		}
		if wait.ValueOr(rt.Config.Defaults.Import.Wait) {
			resp, err := rt.Client.Workflows.ImportProfileAndVerify(context.Background(), req, mlx.ImportProfileWorkflowOptions{
				PollOptions: rt.Config.PollOptions(),
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}
		resp, _, err := rt.Client.Transfers.Import(context.Background(), req)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runImportStatus(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("import status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	importID := fs.String("import-id", "", "import id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx import status --import-id <id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*importID) == "" {
		return errors.New("--import-id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Transfers.ImportStatus(context.Background(), *importID)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runImportStatuses(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("import statuses", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx import statuses")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Transfers.ImportStatuses(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp.Data.Statuses)
	})
}

func runExtension(args []string, global globalOptions) error {
	if len(args) == 0 {
		printExtensionHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return runExtensionList(args[1:], global)
	case "get":
		return runExtensionGet(args[1:], global)
	case "upload":
		return runExtensionUpload(args[1:], global)
	case "create-url":
		return runExtensionCreateURL(args[1:], global)
	case "create-webstore":
		return runExtensionCreateWebStore(args[1:], global)
	case "enable":
		return runExtensionEnable(args[1:], global)
	case "disable":
		return runExtensionDisable(args[1:], global)
	case "usages":
		return runExtensionUsages(args[1:], global)
	case "download":
		return runExtensionDownload(args[1:], global)
	case "delete":
		return runExtensionDelete(args[1:], global)
	case "restore":
		return runExtensionRestore(args[1:], global)
	default:
		printExtensionHelp(os.Stdout)
		return fmt.Errorf("unknown extension subcommand %q", args[0])
	}
}

func runExtensionList(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "object name filter")
	limit := fs.Int("limit", 50, "page size")
	offset := fs.Int("offset", 0, "page offset")
	trashbin := fs.Bool("trashbin", false, "include trashbin objects only")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension list [--name <text>] [--limit <n>] [--offset <n>] [--trashbin]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		tb := *trashbin
		resp, _, err := rt.Client.Resources.ListExtensions(context.Background(), &mlx.ListResourceMetasOptions{
			ObjectName: *name,
			Limit:      *limit,
			Offset:     *offset,
			Trashbin:   &tb,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp.Data.Objects)
	})
}

func runExtensionGet(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "resource id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension get --id <resource-id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.GetMeta(context.Background(), *id)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runExtensionUpload(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension upload", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "path to extension zip")
	storageType := fs.String("storage-type", "", "cloud|local")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension upload --path <zip> [--storage-type <cloud|local>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return errors.New("--path is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		effectiveStorage := strings.TrimSpace(*storageType)
		if effectiveStorage == "" {
			effectiveStorage = rt.Config.Defaults.Extension.StorageType
		}
		resp, _, err := rt.Client.Resources.UploadExtension(context.Background(), &mlx.UploadExtensionRequest{
			ObjectPath:  *path,
			StorageType: effectiveStorage,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runExtensionCreateURL(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension create-url", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rawURL := fs.String("url", "", "download URL")
	browserType := fs.String("browser-type", "", "browser type")
	storageType := fs.String("storage-type", "", "cloud|local")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension create-url --url <download-url> [--browser-type <browser>] [--storage-type <cloud|local>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*rawURL) == "" {
		return errors.New("--url is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.CreateExtensionFromURL(context.Background(), &mlx.CreateExtensionFromURLRequest{
			URL:         *rawURL,
			BrowserType: firstNonEmpty(*browserType, rt.Config.Defaults.Extension.BrowserType),
			StorageType: firstNonEmpty(*storageType, rt.Config.Defaults.Extension.StorageType),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runExtensionCreateWebStore(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension create-webstore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	extensionID := fs.String("extension-id", "", "Chrome Web Store extension id")
	browserType := fs.String("browser-type", "", "browser type")
	storageType := fs.String("storage-type", "", "cloud|local")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension create-webstore --extension-id <id> [--browser-type <browser>] [--storage-type <cloud|local>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*extensionID) == "" {
		return errors.New("--extension-id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.CreateExtensionFromChromeWebStore(context.Background(), &mlx.CreateChromeWebStoreExtensionRequest{
			ExtensionID: *extensionID,
			BrowserType: firstNonEmpty(*browserType, rt.Config.Defaults.Extension.BrowserType),
			StorageType: firstNonEmpty(*storageType, rt.Config.Defaults.Extension.StorageType),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runExtensionEnable(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension enable", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	resourceID := fs.String("id", "", "extension resource id")
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	requireProfileUsageRead := newOptionalBoolFlag(fs, "require-profile-usage-read", "require profile usage read verification")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension enable --id <resource-id> (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--require-profile-usage-read]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*resourceID) == "" {
		return errors.New("--id is required")
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profileUsageReadRequired := requireProfileUsageRead.ValueOr(rt.Config.Defaults.Extension.RequireProfileUsageRead)
		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.EnableExtensionForProfileByName(context.Background(), *profileName, *resourceID, mlx.EnableExtensionForProfileByNameOptions{
				FindOptions:             buildFindOptions(rt.Config, *folderID),
				PollOptions:             rt.Config.PollOptions(),
				RequireProfileUsageRead: profileUsageReadRequired,
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		profile, err := resolveProfile(rt, *profileID, "", *folderID)
		if err != nil {
			return err
		}
		enableResp, _, err := rt.Client.Resources.EnableExtensionForProfiles(context.Background(), *resourceID, &mlx.SetResourceProfilesRequest{
			ProfileIDs: []string{profile.ID},
		})
		if err != nil {
			return err
		}
		usages, _, usageErr := rt.Client.Resources.ObjectProfileUsages(context.Background(), *resourceID)
		profileUsages, _, profileUsageErr := rt.Client.Resources.ProfileExtensionUsages(context.Background(), profile.ID)
		if profileUsageErr != nil && profileUsageReadRequired {
			return profileUsageErr
		}
		return emit(rt, map[string]any{
			"profile":         profile,
			"enable_response": enableResp,
			"object_usages":   usages,
			"profile_usages":  profileUsages,
			"profile_error":   errorString(profileUsageErr),
			"usage_error":     errorString(usageErr),
		})
	})
}

func runExtensionDisable(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension disable", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	resourceID := fs.String("id", "", "extension resource id")
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension disable --id <resource-id> (--profile-id <id> | --profile-name <name>) [--folder-id <id>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*resourceID) == "" {
		return errors.New("--id is required")
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Resources.DisableExtensionForProfiles(context.Background(), *resourceID, &mlx.SetResourceProfilesRequest{
			ProfileIDs: []string{profile.ID},
		})
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{
			"profile":          profile,
			"disable_response": resp,
		})
	})
}

func runExtensionUsages(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension usages", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	resourceID := fs.String("id", "", "extension resource id")
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension usages --id <resource-id> | (--profile-id <id> | --profile-name <name>)")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	if strings.TrimSpace(*resourceID) == "" && strings.TrimSpace(*profileID) == "" && strings.TrimSpace(*profileName) == "" {
		return errors.New("one of --id, --profile-id, or --profile-name is required")
	}
	if strings.TrimSpace(*resourceID) != "" && (strings.TrimSpace(*profileID) != "" || strings.TrimSpace(*profileName) != "") {
		return errors.New("--id cannot be combined with --profile-id or --profile-name")
	}
	if strings.TrimSpace(*profileID) != "" && strings.TrimSpace(*profileName) != "" {
		return errors.New("--profile-id and --profile-name are mutually exclusive")
	}

	return withRuntime(global, func(rt *Runtime) error {
		if strings.TrimSpace(*resourceID) != "" {
			resp, _, err := rt.Client.Resources.ObjectProfileUsages(context.Background(), *resourceID)
			if err != nil {
				return err
			}
			return emit(rt, resp.Data)
		}
		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Resources.ProfileExtensionUsages(context.Background(), profile.ID)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runExtensionDownload(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension download", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "resource id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension download --id <resource-id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.Download(context.Background(), *id)
		if err != nil {
			return err
		}
		return emit(rt, map[string]any{"path": resp.Path})
	})
}

func runExtensionDelete(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "resource id")
	permanently := fs.Bool("permanently", false, "permanent delete")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension delete --id <resource-id> [--permanently]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.Delete(context.Background(), *id, *permanently)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runExtensionRestore(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "resource id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx extension restore --id <resource-id>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.Restore(context.Background(), *id)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runCookies(args []string, global globalOptions) error {
	if len(args) == 0 {
		printCookiesHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "websites":
		return runCookiesWebsites(args[1:], global)
	case "list":
		return runCookiesList(args[1:], global)
	case "metadata":
		return runCookiesMetadata(args[1:], global)
	case "import":
		return runCookiesImport(args[1:], global)
	case "export":
		return runCookiesExport(args[1:], global)
	case "seed":
		return runCookiesSeed(args[1:], global)
	default:
		printCookiesHelp(os.Stdout)
		return fmt.Errorf("unknown cookies subcommand %q", args[0])
	}
}

func runCookiesWebsites(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies websites", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies websites")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Cookies.ListWebsites(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp.Data)
	})
}

func runCookiesList(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies list --profile-id <id> | --profile-name <name>")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Cookies.List(context.Background(), profile.ID)
		if err != nil {
			return err
		}
		return emit(rt, resp.Data.Cookies)
	})
}

func runCookiesMetadata(args []string, global globalOptions) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies metadata <create|update> [flags]")
		return nil
	}

	switch args[0] {
	case "create":
		return runCookiesMetadataCreate(args[1:], global)
	case "update":
		return runCookiesMetadataUpdate(args[1:], global)
	default:
		return fmt.Errorf("unknown cookies metadata subcommand %q", args[0])
	}
}

func runCookiesMetadataCreate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies metadata create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	targetWebsite := fs.String("target-website", "", "target website key")
	strict := fs.Bool("strict", false, "enable strict mode")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies metadata create (--profile-id <id> | --profile-name <name>) --target-website <key> [--strict]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}
	if strings.TrimSpace(*targetWebsite) == "" {
		return errors.New("--target-website is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Cookies.CreateMetadata(context.Background(), &mlx.CreateCookiesMetadataRequest{
			ProfileID:     profile.ID,
			TargetWebsite: *targetWebsite,
			StrictMode:    *strict,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runCookiesMetadataUpdate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies metadata update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	targetWebsite := fs.String("target-website", "", "target website key")
	additionalWebsite := fs.String("additional-website", "", "additional website key")
	strict := fs.Bool("strict", false, "enable strict mode")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies metadata update (--profile-id <id> | --profile-name <name>) --target-website <key> [--additional-website <key>] [--strict]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}
	if strings.TrimSpace(*targetWebsite) == "" {
		return errors.New("--target-website is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Cookies.UpdateMetadata(context.Background(), &mlx.UpdateCookiesMetadataRequest{
			ProfileID:         profile.ID,
			TargetWebsite:     *targetWebsite,
			AdditionalWebsite: *additionalWebsite,
			StrictMode:        *strict,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runCookiesImport(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "launcher folder id")
	advanced := newOptionalBoolFlag(fs, "advanced", "import advanced pre-made cookies")
	strict := newOptionalBoolFlag(fs, "strict", "enable strict mode")
	cookiesFile := fs.String("cookies-file", "", "path to BrowserCookie array JSON")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies import (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--advanced] [--strict] [--cookies-file <cookies.json>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	var cookies []mlx.BrowserCookie
	var err error
	if strings.TrimSpace(*cookiesFile) != "" {
		cookies, err = readCookiesFile(*cookiesFile)
		if err != nil {
			return err
		}
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, "")
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Cookies.Import(context.Background(), &mlx.CookieImportRequest{
			ProfileID:             profile.ID,
			FolderID:              firstNonEmpty(*folderID, profile.FolderID),
			ImportAdvancedCookies: advanced.ValueOr(rt.Config.Defaults.Cookies.ImportAdvancedCookies),
			Cookies:               cookies,
			StrictMode:            strict.ValueOr(rt.Config.Defaults.Cookies.StrictMode),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runCookiesExport(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "launcher folder id")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies export (--profile-id <id> | --profile-name <name>) [--folder-id <id>]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, "")
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Cookies.Export(context.Background(), &mlx.CookieExportRequest{
			ProfileID: profile.ID,
			FolderID:  firstNonEmpty(*folderID, profile.FolderID),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runCookiesSeed(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("cookies seed", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "launcher folder id")
	targetWebsite := fs.String("target-website", "", "target website key")
	additionalWebsite := fs.String("additional-website", "", "additional website key")
	createMetadataIfMissing := newOptionalBoolFlag(fs, "create-metadata-if-missing", "create metadata if missing")
	advanced := newOptionalBoolFlag(fs, "advanced", "import advanced pre-made cookies")
	strict := newOptionalBoolFlag(fs, "strict", "enable strict mode")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx cookies seed (--profile-id <id> | --profile-name <name>) [--folder-id <id>] --target-website <key> [--additional-website <key>] [--create-metadata-if-missing] [--advanced] [--strict]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		target := strings.TrimSpace(*targetWebsite)
		if target == "" {
			target = strings.TrimSpace(rt.Config.Defaults.Cookies.TargetWebsite)
		}
		if target == "" {
			return errors.New("--target-website is required")
		}
		additional := strings.TrimSpace(*additionalWebsite)
		if additional == "" {
			additional = strings.TrimSpace(rt.Config.Defaults.Cookies.AdditionalWebsite)
		}

		profile, err := resolveProfile(rt, *profileID, *profileName, "")
		if err != nil {
			return err
		}
		resp, err := rt.Client.Cookies.SeedProfileCookies(context.Background(), mlx.SeedProfileCookiesOptions{
			ProfileID:               profile.ID,
			FolderID:                firstNonEmpty(*folderID, profile.FolderID),
			TargetWebsite:           target,
			AdditionalWebsite:       additional,
			CreateMetadataIfMissing: createMetadataIfMissing.ValueOr(rt.Config.Defaults.Cookies.CreateMetadataIfMissing),
			StrictMode:              strict.ValueOr(rt.Config.Defaults.Cookies.StrictMode),
			ImportAdvancedCookies:   advanced.ValueOr(rt.Config.Defaults.Cookies.ImportAdvancedCookies),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProxy(args []string, global globalOptions) error {
	if len(args) == 0 {
		printProxyHelp(os.Stdout)
		return nil
	}

	switch args[0] {
	case "usage":
		return runProxyUsage(args[1:], global)
	case "generate":
		return runProxyGenerate(args[1:], global)
	case "assign":
		return runProxyAssign(args[1:], global)
	default:
		printProxyHelp(os.Stdout)
		return fmt.Errorf("unknown proxy subcommand %q", args[0])
	}
}

func runProxyUsage(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("proxy usage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx proxy usage")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Proxies.GetUsage(context.Background())
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProxyGenerate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("proxy generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	country := fs.String("country", "", "country code")
	region := fs.String("region", "", "region name")
	city := fs.String("city", "", "city name")
	protocol := fs.String("protocol", "", "socks5|http")
	sessionType := fs.String("session-type", "", "sticky|rotating")
	ipTTL := fs.Int("ip-ttl", 0, "IPTTL value")
	count := fs.Int("count", 1, "number of proxies")
	strict := fs.Bool("strict", false, "enable strict mode")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx proxy generate [--country <code>] [--region <name>] [--city <name>] [--protocol <socks5|http>] [--session-type <sticky|rotating>] [--ip-ttl <seconds>] [--count <n>] [--strict]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		req := buildGenerateProxyRequest(rt.Config, *country, *region, *city, *protocol, *sessionType, *ipTTL, *count, *strict)
		resp, _, err := rt.Client.Proxies.Generate(context.Background(), req)
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runProxyAssign(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("proxy assign", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	country := fs.String("country", "", "country code")
	region := fs.String("region", "", "region name")
	city := fs.String("city", "", "city name")
	protocol := fs.String("protocol", "", "socks5|http")
	sessionType := fs.String("session-type", "", "sticky|rotating")
	ipTTL := fs.Int("ip-ttl", 0, "IPTTL value")
	strict := fs.Bool("strict", false, "enable strict mode")
	preferSOCKS5 := newOptionalBoolFlag(fs, "prefer-socks5", "prefer socks5 when protocol is not set")
	saveTraffic := newOptionalBoolFlag(fs, "save-traffic", "save traffic in generated profile proxy")
	patchProfile := newOptionalBoolFlag(fs, "patch-profile", "patch profile with generated proxy")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx proxy assign (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--country <code>] [--region <name>] [--city <name>] [--protocol <socks5|http>] [--session-type <sticky|rotating>] [--ip-ttl <seconds>] [--strict] [--prefer-socks5] [--save-traffic] [--patch-profile]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(*profileID, *profileName, "--profile-id", "--profile-name"); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		effectivePatchProfile := patchProfile.ValueOr(rt.Config.Defaults.Proxy.PatchProfile)
		generateReq := mlx.GenerateProfileProxyRequest{
			GenerateProxyRequest: *buildGenerateProxyRequest(rt.Config, *country, *region, *city, *protocol, *sessionType, *ipTTL, 1, *strict),
			PreferSOCKS5:         preferSOCKS5.ValueOr(rt.Config.Defaults.Proxy.PreferSOCKS5),
			SaveTraffic:          saveTraffic.ValueOr(rt.Config.Defaults.Proxy.SaveTraffic),
		}

		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.GenerateProfileProxyByName(context.Background(), *profileName, mlx.GenerateProfileProxyByNameOptions{
				FindOptions:     buildFindOptions(rt.Config, *folderID),
				GenerateOptions: generateReq,
				PatchProfile:    effectivePatchProfile,
			})
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		profile, err := resolveProfile(rt, *profileID, "", *folderID)
		if err != nil {
			return err
		}
		generated, err := rt.Client.Proxies.GenerateProfileProxy(context.Background(), &generateReq)
		if err != nil {
			return err
		}

		var patchResp *mlx.EmptyDataResponse
		if effectivePatchProfile {
			patchResp, _, err = rt.Client.Profiles.Patch(context.Background(), &mlx.PatchProfileRequest{
				ProfileID: profile.ID,
				Proxy:     generated.ProfileProxy,
			})
			if err != nil {
				return err
			}
		}

		return emit(rt, map[string]any{
			"profile":        profile,
			"connection":     generated.Connection,
			"profile_proxy":  generated.ProfileProxy,
			"usage":          generated.Usage,
			"patch_response": patchResp,
		})
	})
}

func emit(rt *Runtime, value any) error {
	return emitFormatted(os.Stdout, rt.Config.Output.Format, rt.Config.Output.Pretty, value)
}

func emitWithGlobal(global globalOptions, value any) error {
	cfg := DefaultConfig()
	if override := strings.ToLower(strings.TrimSpace(global.Output)); override != "" {
		cfg.Output.Format = override
		cfg = cfg.Normalize()
		if err := cfg.Validate(); err != nil {
			return err
		}
	}
	return emitFormatted(os.Stdout, cfg.Output.Format, cfg.Output.Pretty, value)
}

func emitFormatted(w io.Writer, format string, pretty bool, value any) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", outputFormatTable:
		return renderTable(w, value)
	case outputFormatJSON:
		return renderJSON(w, value, pretty)
	case outputFormatYAML:
		return renderYAML(w, value)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func renderJSON(w io.Writer, value any, pretty bool) error {
	var (
		body []byte
		err  error
	)
	if pretty {
		body, err = json.MarshalIndent(value, "", "  ")
	} else {
		body, err = json.Marshal(value)
	}
	if err != nil {
		return err
	}
	if _, err := w.Write(append(body, '\n')); err != nil {
		return err
	}
	return nil
}

func renderYAML(w io.Writer, value any) error {
	generic, err := normalizeValue(value)
	if err != nil {
		return err
	}
	if err := writeYAMLNode(w, generic, 0, true); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

func renderTable(w io.Writer, value any) error {
	generic, err := normalizeValue(value)
	if err != nil {
		return err
	}

	switch v := generic.(type) {
	case nil:
		_, err := fmt.Fprintln(w, "(empty)")
		return err
	case []any:
		return renderTableSlice(w, v)
	case map[string]any:
		return renderTableMap(w, v)
	default:
		_, err := fmt.Fprintln(w, formatScalar(v))
		return err
	}
}

func renderTableSlice(w io.Writer, rows []any) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "(empty)")
		return err
	}

	allMaps := true
	for _, row := range rows {
		if _, ok := row.(map[string]any); !ok {
			allMaps = false
			break
		}
	}
	if !allMaps {
		for _, row := range rows {
			if _, err := fmt.Fprintf(w, "- %s\n", formatCell(row)); err != nil {
				return err
			}
		}
		return nil
	}

	keys := make(map[string]struct{})
	for _, row := range rows {
		for key := range row.(map[string]any) {
			keys[key] = struct{}{}
		}
	}
	header := make([]string, 0, len(keys))
	for key := range keys {
		header = append(header, key)
	}
	sort.Strings(header)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(header, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		m := row.(map[string]any)
		cells := make([]string, 0, len(header))
		for _, key := range header {
			cells = append(cells, formatCell(m[key]))
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func renderTableMap(w io.Writer, values map[string]any) error {
	if len(values) == 0 {
		_, err := fmt.Fprintln(w, "(empty)")
		return err
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, key := range keys {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", key, formatCell(values[key])); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func normalizeValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(body, &generic); err != nil {
		return nil, err
	}
	return generic, nil
}

func writeYAMLNode(w io.Writer, node any, indent int, topLevel bool) error {
	prefix := strings.Repeat("  ", indent)
	switch v := node.(type) {
	case nil:
		_, err := io.WriteString(w, prefix+"null")
		return err
	case map[string]any:
		if len(v) == 0 {
			_, err := io.WriteString(w, prefix+"{}")
			return err
		}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			if !topLevel || i > 0 {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
			if isScalarYAML(v[key]) {
				if _, err := fmt.Fprintf(w, "%s%s: %s", prefix, key, formatYAMLScalar(v[key])); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s%s:", prefix, key); err != nil {
				return err
			}
			if err := writeYAMLNode(w, v[key], indent+1, false); err != nil {
				return err
			}
		}
		return nil
	case []any:
		if len(v) == 0 {
			_, err := io.WriteString(w, prefix+"[]")
			return err
		}
		for i, item := range v {
			if !topLevel || i > 0 {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
			if isScalarYAML(item) {
				if _, err := fmt.Fprintf(w, "%s- %s", prefix, formatYAMLScalar(item)); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%s-", prefix); err != nil {
				return err
			}
			if err := writeYAMLNode(w, item, indent+1, false); err != nil {
				return err
			}
		}
		return nil
	default:
		_, err := io.WriteString(w, prefix+formatYAMLScalar(v))
		return err
	}
}

func isScalarYAML(v any) bool {
	switch v.(type) {
	case nil, string, bool, float64, int, int64, uint64:
		return true
	default:
		return false
	}
}

func formatYAMLScalar(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		body, _ := json.Marshal(x)
		return string(body)
	default:
		return fmt.Sprint(x)
	}
}

func formatCell(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case map[string]any, []any:
		body, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(body)
	default:
		return fmt.Sprint(v)
	}
}

func formatScalar(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func readJSONFile(path string, out any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("%s contains multiple JSON values", path)
		}
		return fmt.Errorf("decode trailing content in %s: %w", path, err)
	}
	return nil
}

func readCookiesFile(path string) ([]mlx.BrowserCookie, error) {
	var cookies []mlx.BrowserCookie
	if err := readJSONFile(path, &cookies); err != nil {
		return nil, err
	}
	return cookies, nil
}

type profileTemplateDocument struct {
	Name       string                   `json:"name"`
	MainParams mlx.CreateProfileRequest `json:"mainParams"`
}

func loadProfileTemplate(metaInfo, path string) (*profileTemplateDocument, error) {
	var pathErr error

	if strings.TrimSpace(path) != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			pathErr = fmt.Errorf("read template %s: %w", path, err)
		} else {
			var doc profileTemplateDocument
			if err := json.Unmarshal(body, &doc); err != nil {
				pathErr = fmt.Errorf("decode template %s: %w", path, err)
			} else if profileTemplateHasUsableMainParams(&doc) || strings.TrimSpace(doc.Name) != "" {
				return &doc, nil
			}
		}
	}

	if strings.TrimSpace(metaInfo) != "" {
		var doc profileTemplateDocument
		if err := json.Unmarshal([]byte(metaInfo), &doc); err != nil {
			return nil, fmt.Errorf("decode template meta_info: %w", err)
		}
		if profileTemplateHasUsableMainParams(&doc) || strings.TrimSpace(doc.Name) != "" {
			return &doc, nil
		}
		return nil, errors.New("template meta_info does not contain usable mainParams")
	}

	if pathErr != nil {
		return nil, pathErr
	}
	return nil, errors.New("template content is empty")
}

func profileTemplateHasUsableMainParams(doc *profileTemplateDocument) bool {
	if doc == nil {
		return false
	}

	main := doc.MainParams
	return strings.TrimSpace(main.Name) != "" ||
		strings.TrimSpace(main.BrowserType) != "" ||
		strings.TrimSpace(main.FolderID) != "" ||
		strings.TrimSpace(main.OSType) != "" ||
		main.CoreVersion != 0 ||
		main.CoreMinorVersion != 0 ||
		main.AutoUpdateCore != nil ||
		main.Times != 0 ||
		strings.TrimSpace(main.Notes) != "" ||
		main.Parameters != nil ||
		len(main.Tags) != 0
}

func buildCreateProfileRequestFromTemplate(doc *profileTemplateDocument, name, folderID string, localOverride *bool) (*mlx.CreateProfileRequest, error) {
	if doc == nil {
		return nil, errors.New("template document is required")
	}

	req := doc.MainParams
	if strings.TrimSpace(name) != "" {
		req.Name = name
	} else if strings.TrimSpace(req.Name) == "" {
		req.Name = strings.TrimSpace(doc.Name)
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("template does not contain a usable profile name; pass --name")
	}

	if strings.TrimSpace(folderID) != "" {
		req.FolderID = folderID
	}
	if strings.TrimSpace(req.FolderID) == "" {
		return nil, errors.New("template does not contain a folder id and no --folder-id/default folder id was resolved")
	}

	if localOverride != nil {
		if req.Parameters == nil {
			req.Parameters = &mlx.ProfileParameters{}
		}
		if req.Parameters.Storage == nil {
			req.Parameters.Storage = &mlx.Storage{}
		}
		req.Parameters.Storage.IsLocal = *localOverride
	}

	return &req, nil
}

func resolveFolderID(rt *Runtime, explicit string) (string, error) {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return trimmed, nil
	}
	if trimmed := strings.TrimSpace(rt.Config.Defaults.Folder.ID); trimmed != "" {
		return trimmed, nil
	}

	defaultName := strings.TrimSpace(rt.Config.Defaults.Folder.Name)
	if defaultName == "" {
		return "", nil
	}

	resp, _, err := rt.Client.Folders.List(context.Background())
	if err != nil {
		return "", err
	}
	for _, folder := range resp.Data.Folders {
		if strings.EqualFold(strings.TrimSpace(folder.Name), defaultName) {
			return folder.FolderID, nil
		}
	}

	return "", nil
}

func resolveProfile(rt *Runtime, profileID, profileName, folderID string) (*resolvedProfile, error) {
	if strings.TrimSpace(profileID) != "" {
		meta, _, err := rt.Client.Profiles.GetMeta(context.Background(), profileID)
		if err != nil {
			return nil, err
		}
		return &resolvedProfile{
			ID:       meta.ID,
			Name:     meta.Name,
			FolderID: meta.FolderID,
			Meta:     meta,
		}, nil
	}

	verified, err := rt.Client.Workflows.FindProfileByNameVerified(context.Background(), profileName, mlx.FindProfileByNameVerifiedOptions{
		FindOptions: buildFindOptions(rt.Config, folderID),
	})
	if err != nil {
		return nil, err
	}
	return &resolvedProfile{
		ID:       verified.Profile.ID,
		Name:     verified.Profile.Name,
		FolderID: verified.Profile.FolderID,
		Profile:  verified.Profile,
		Meta:     verified.Meta,
	}, nil
}

func buildFindOptions(cfg Config, folderID string) *mlx.FindProfileOptions {
	find := &mlx.FindProfileOptions{
		StorageType: cfg.Defaults.Profile.StorageType,
		FolderID:    firstNonEmpty(folderID, cfg.Defaults.Folder.ID),
	}
	if strings.TrimSpace(find.StorageType) == "" {
		find.StorageType = storageTypeAll
	}
	if find.StorageType == storageTypeAll && strings.TrimSpace(find.FolderID) == "" {
		return nil
	}
	return find
}

func buildGenerateProxyRequest(cfg Config, country, region, city, protocol, sessionType string, ipTTL, count int, strict bool) *mlx.GenerateProxyRequest {
	effectiveCountry := strings.TrimSpace(country)
	if effectiveCountry == "" {
		effectiveCountry = strings.TrimSpace(cfg.Defaults.Proxy.Country)
	}
	effectiveRegion := strings.TrimSpace(region)
	if effectiveRegion == "" {
		effectiveRegion = strings.TrimSpace(cfg.Defaults.Proxy.Region)
	}
	effectiveCity := strings.TrimSpace(city)
	if effectiveCity == "" {
		effectiveCity = strings.TrimSpace(cfg.Defaults.Proxy.City)
	}
	effectiveProtocol := strings.TrimSpace(protocol)
	if effectiveProtocol == "" {
		effectiveProtocol = cfg.Defaults.Proxy.Protocol
	}
	effectiveSessionType := strings.TrimSpace(sessionType)
	if effectiveSessionType == "" {
		effectiveSessionType = cfg.Defaults.Proxy.SessionType
	}
	if count <= 0 {
		count = 1
	}
	return &mlx.GenerateProxyRequest{
		Country:     effectiveCountry,
		Region:      effectiveRegion,
		City:        effectiveCity,
		Protocol:    mlx.ProxyProtocol(effectiveProtocol),
		SessionType: mlx.ProxySessionType(effectiveSessionType),
		IPTTL:       ipTTL,
		Count:       count,
		StrictMode:  strict,
	}
}

func waitForStoppedStatus(ctx context.Context, rt *Runtime, profileID string) (*mlx.ProfileRuntimeStatusResponse, error) {
	opts := rt.Config.PollOptions()
	deadline := time.Now().Add(opts.Timeout)
	interval := opts.InitialInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if opts.MaxInterval <= 0 {
		opts.MaxInterval = interval
	}
	if opts.Multiplier <= 1 {
		opts.Multiplier = 1.5
	}

	var last *mlx.ProfileRuntimeStatusResponse
	for {
		resp, _, err := rt.Client.Launcher.Status(ctx, profileID)
		if err == nil {
			last = resp
			status := strings.ToLower(strings.TrimSpace(resp.Data.Status))
			if status == "" || !strings.Contains(status, "running") {
				return resp, nil
			}
		}
		if !time.Now().Before(deadline) {
			if last != nil {
				return nil, fmt.Errorf("profile %s did not reach stopped status before timeout, last status=%s", profileID, last.Data.Status)
			}
			return nil, fmt.Errorf("profile %s did not reach stopped status before timeout", profileID)
		}
		if err := sleepContext(ctx, interval); err != nil {
			return nil, err
		}
		interval = nextInterval(interval, opts.Multiplier, opts.MaxInterval)
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func nextInterval(current time.Duration, multiplier float64, max time.Duration) time.Duration {
	next := time.Duration(float64(current) * multiplier)
	if next < current {
		next = current
	}
	if max > 0 && next > max {
		return max
	}
	return next
}

func validateSelector(id, name, idFlag, nameFlag string) error {
	if strings.TrimSpace(id) == "" && strings.TrimSpace(name) == "" {
		return fmt.Errorf("one of %s or %s is required", idFlag, nameFlag)
	}
	if strings.TrimSpace(id) != "" && strings.TrimSpace(name) != "" {
		return fmt.Errorf("%s and %s are mutually exclusive", idFlag, nameFlag)
	}
	return nil
}

func requireNoExtraArgs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
	}
	return nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isAlreadyStoppedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "already stopped")
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
