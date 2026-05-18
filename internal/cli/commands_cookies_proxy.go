package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mlx "mlx-go-sdk"
)

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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies list --profile-id <id> | --profile-name <name>",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies metadata <create|update> [flags]",
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies metadata create (--profile-id <id> | --profile-name <name>) --target-website <key> [--strict]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
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
		resp, _, err := rt.Client.Cookies.CreateMetadata(
			context.Background(),
			&mlx.CreateCookiesMetadataRequest{
				ProfileID:     profile.ID,
				TargetWebsite: *targetWebsite,
				StrictMode:    *strict,
			},
		)
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
	additionalWebsite := fs.String(
		"additional-website",
		"",
		"additional website key",
	)
	strict := fs.Bool("strict", false, "enable strict mode")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies metadata update (--profile-id <id> | --profile-name <name>) --target-website <key> [--additional-website <key>] [--strict]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
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
		resp, _, err := rt.Client.Cookies.UpdateMetadata(
			context.Background(),
			&mlx.UpdateCookiesMetadataRequest{
				ProfileID:         profile.ID,
				TargetWebsite:     *targetWebsite,
				AdditionalWebsite: *additionalWebsite,
				StrictMode:        *strict,
			},
		)
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
	advanced := newOptionalBoolFlag(
		fs,
		"advanced",
		"import advanced pre-made cookies",
	)
	strict := newOptionalBoolFlag(fs, "strict", "enable strict mode")
	cookiesFile := fs.String(
		"cookies-file",
		"",
		"path to BrowserCookie array JSON",
	)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies import (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--advanced] [--strict] [--cookies-file <cookies.json>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
		return err
	}

	var (
		cookies []mlx.BrowserCookie
		err     error
	)
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
		resp, _, err := rt.Client.Cookies.Import(
			context.Background(),
			&mlx.CookieImportRequest{
				ProfileID: profile.ID,
				FolderID:  firstNonEmpty(*folderID, profile.FolderID),
				ImportAdvancedCookies: advanced.ValueOr(
					rt.Config.Defaults.Cookies.ImportAdvancedCookies,
				),
				Cookies:    cookies,
				StrictMode: strict.ValueOr(rt.Config.Defaults.Cookies.StrictMode),
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies export (--profile-id <id> | --profile-name <name>) [--folder-id <id>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		profile, err := resolveProfile(rt, *profileID, *profileName, "")
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Cookies.Export(
			context.Background(),
			&mlx.CookieExportRequest{
				ProfileID: profile.ID,
				FolderID:  firstNonEmpty(*folderID, profile.FolderID),
			},
		)
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
	additionalWebsite := fs.String(
		"additional-website",
		"",
		"additional website key",
	)
	createMetadataIfMissing := newOptionalBoolFlag(
		fs,
		"create-metadata-if-missing",
		"create metadata if missing",
	)
	advanced := newOptionalBoolFlag(
		fs,
		"advanced",
		"import advanced pre-made cookies",
	)
	strict := newOptionalBoolFlag(fs, "strict", "enable strict mode")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx cookies seed (--profile-id <id> | --profile-name <name>) [--folder-id <id>] --target-website <key> [--additional-website <key>] [--create-metadata-if-missing] [--advanced] [--strict]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
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
			additional = strings.TrimSpace(
				rt.Config.Defaults.Cookies.AdditionalWebsite,
			)
		}

		profile, err := resolveProfile(rt, *profileID, *profileName, "")
		if err != nil {
			return err
		}
		resp, err := rt.Client.Cookies.SeedProfileCookies(
			context.Background(),
			mlx.SeedProfileCookiesOptions{
				ProfileID:         profile.ID,
				FolderID:          firstNonEmpty(*folderID, profile.FolderID),
				TargetWebsite:     target,
				AdditionalWebsite: additional,
				CreateMetadataIfMissing: createMetadataIfMissing.ValueOr(
					rt.Config.Defaults.Cookies.CreateMetadataIfMissing,
				),
				StrictMode: strict.ValueOr(rt.Config.Defaults.Cookies.StrictMode),
				ImportAdvancedCookies: advanced.ValueOr(
					rt.Config.Defaults.Cookies.ImportAdvancedCookies,
				),
			},
		)
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
	case "validate":
		return runProxyValidate(args[1:], global)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx proxy generate [--country <code>] [--region <name>] [--city <name>] [--protocol <socks5|http>] [--session-type <sticky|rotating>] [--ip-ttl <seconds>] [--count <n>] [--strict]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		req := buildGenerateProxyRequest(
			rt.Config,
			*country,
			*region,
			*city,
			*protocol,
			*sessionType,
			*ipTTL,
			*count,
			*strict,
		)
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
	preferSOCKS5 := newOptionalBoolFlag(
		fs,
		"prefer-socks5",
		"prefer socks5 when protocol is not set",
	)
	saveTraffic := newOptionalBoolFlag(
		fs,
		"save-traffic",
		"save traffic in generated profile proxy",
	)
	patchProfile := newOptionalBoolFlag(
		fs,
		"patch-profile",
		"patch profile with generated proxy",
	)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx proxy assign (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--country <code>] [--region <name>] [--city <name>] [--protocol <socks5|http>] [--session-type <sticky|rotating>] [--ip-ttl <seconds>] [--strict] [--prefer-socks5] [--save-traffic] [--patch-profile]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if err := validateSelector(
		*profileID,
		*profileName,
		"--profile-id",
		"--profile-name",
	); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		effectivePatchProfile := patchProfile.ValueOr(
			rt.Config.Defaults.Proxy.PatchProfile,
		)
		generateReq := mlx.GenerateProfileProxyRequest{
			GenerateProxyRequest: *buildGenerateProxyRequest(
				rt.Config,
				*country,
				*region,
				*city,
				*protocol,
				*sessionType,
				*ipTTL,
				1,
				*strict,
			),
			PreferSOCKS5: preferSOCKS5.ValueOr(
				rt.Config.Defaults.Proxy.PreferSOCKS5,
			),
			SaveTraffic: saveTraffic.ValueOr(
				rt.Config.Defaults.Proxy.SaveTraffic,
			),
		}

		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.GenerateProfileProxyByName(
				context.Background(),
				*profileName,
				mlx.GenerateProfileProxyByNameOptions{
					FindOptions:     buildFindOptions(rt.Config, *folderID),
					GenerateOptions: generateReq,
					PatchProfile:    effectivePatchProfile,
				},
			)
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		profile, err := resolveProfile(rt, *profileID, "", *folderID)
		if err != nil {
			return err
		}
		generated, err := rt.Client.Proxies.GenerateProfileProxy(
			context.Background(),
			&generateReq,
		)
		if err != nil {
			return err
		}

		var patchResp *mlx.EmptyDataResponse
		if effectivePatchProfile {
			patchResp, _, err = rt.Client.Profiles.Patch(
				context.Background(),
				&mlx.PatchProfileRequest{
					ProfileID: profile.ID,
					Proxy:     generated.ProfileProxy,
				},
			)
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

func runProxyValidate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("proxy validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	proxyType := fs.String("type", "", "proxy type: http|socks5")
	host := fs.String("host", "", "proxy host")
	port := fs.Int("port", 0, "proxy port")
	username := fs.String("username", "", "proxy username")
	password := fs.String("password", "", "proxy password")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx proxy validate --type TYPE --host HOST --port PORT [--username USER] [--password PASS]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*proxyType) == "" {
		return errors.New("--type is required")
	}
	if strings.TrimSpace(*host) == "" {
		return errors.New("--host is required")
	}
	if *port <= 0 {
		return errors.New("--port must be greater than 0")
	}
	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Launcher.ValidateProxy(context.Background(), &mlx.ValidateProxyRequest{
			Type:     *proxyType,
			Host:     *host,
			Port:     *port,
			Username: *username,
			Password: *password,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}
