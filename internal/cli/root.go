package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
