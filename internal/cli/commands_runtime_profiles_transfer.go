package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mlx "github.com/minskyagenda0708-cmd/mlx-go-sdk"
)

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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx launcher status --profile-id <id> | --profile-name <name>",
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
		resp, _, err := rt.Client.Launcher.Status(
			context.Background(),
			profile.ID,
		)
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
	automationType := fs.String(
		"automation-type",
		"",
		"selenium|playwright|puppeteer|rod",
	)
	headless := newOptionalBoolFlag(fs, "headless", "start headless")
	strict := newOptionalBoolFlag(fs, "strict", "enable strict mode")
	wait := newOptionalBoolFlag(fs, "wait", "wait for running status")
	skipProxyCheck := fs.Bool("skip-proxy-check", false, "DANGEROUS: skip the pre-launch proxy health check")
	proxyThreshold := fs.Int("proxy-threshold-ms", 0, "override proxy latency threshold (ms)")
	proxyHardCap := fs.Int("proxy-hard-cap-ms", 0, "override proxy latency hard cap (ms)")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx launcher start --profile-id <id> | --profile-name <name> [--folder-id <id>] [--automation-type <type>] [--headless] [--strict] [--wait] [--skip-proxy-check] [--proxy-threshold-ms <ms>] [--proxy-hard-cap-ms <ms>]",
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
			// Resolve the name to a concrete profile ID so the fail-closed
			// proxy check runs before StartProfileByName (cannot be bypassed
			// via --profile-name).
			named, err := resolveProfile(rt, "", *profileName, *folderID)
			if err != nil {
				return err
			}
			if err := ensureProxyBeforeStart(rt, named.ID, *skipProxyCheck, *proxyThreshold, *proxyHardCap); err != nil {
				return err
			}
			resp, err := rt.Client.Workflows.StartProfileByName(
				context.Background(),
				*profileName,
				mlx.StartProfileByNameOptions{
					FindOptions:    buildFindOptions(rt.Config, *folderID),
					StartOptions:   opts,
					WaitForRunning: effectiveWait,
					PollOptions:    rt.Config.PollOptions(),
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
		started, err := startProfileWithProxyCheck(
			rt,
			firstNonEmpty(*folderID, profile.FolderID),
			profile.ID,
			opts,
			effectiveWait,
			*skipProxyCheck,
			*proxyThreshold,
			*proxyHardCap,
		)
		if err != nil {
			return err
		}
		if started.RuntimeStatus == nil {
			return emit(rt, started.StartResponse)
		}

		return emit(rt, map[string]any{
			"profile":         profile,
			"start_response":  started.StartResponse,
			"runtime_status":  started.RuntimeStatus,
			"automation_type": startAutomation,
		})
	})
}

// startedProfile carries the launcher responses produced by
// startProfileWithProxyCheck so callers can build their own emit shape.
// RuntimeStatus is nil unless the caller requested a wait for running status.
type startedProfile struct {
	StartResponse *mlx.StartProfileResponse
	RuntimeStatus *mlx.ProfileRuntimeStatusResponse
}

// startProfileWithProxyCheck runs the fail-closed proxy continuity check for
// profileID and then launches it via the launcher, optionally waiting for the
// running status. The proxy check always runs first (unless skipProxyCheck is
// set), so an unhealthy proxy prevents the launch (fail-closed). It is shared
// by `launcher start` (id branch) and `profile create --start`.
func startProfileWithProxyCheck(
	rt *Runtime,
	folderID, profileID string,
	opts mlx.StartProfileOptions,
	wait, skipProxyCheck bool,
	thresholdOverride, hardCapOverride int,
) (*startedProfile, error) {
	if err := ensureProxyBeforeStart(rt, profileID, skipProxyCheck, thresholdOverride, hardCapOverride); err != nil {
		return nil, err
	}
	startResp, _, err := rt.Client.Launcher.Start(
		context.Background(),
		folderID,
		profileID,
		opts,
	)
	if err != nil {
		return nil, err
	}
	if !wait {
		return &startedProfile{StartResponse: startResp}, nil
	}

	statusResp, _, err := rt.Client.Launcher.WaitForRunning(
		context.Background(),
		profileID,
		rt.Config.PollOptions(),
	)
	if err != nil {
		return nil, err
	}

	return &startedProfile{StartResponse: startResp, RuntimeStatus: statusResp}, nil
}

// ensureProxyBeforeStart runs the fail-closed proxy continuity check for the
// given profile before it is launched. When proxy continuity is disabled or
// skip is set, it is a no-op. On ANY proxy-check error it returns the error so
// the caller does NOT start the profile (fail-closed). If a healthier proxy is
// chosen, it is patched onto the profile before returning.
func ensureProxyBeforeStart(rt *Runtime, profileID string, skip bool, thresholdOverride, hardCapOverride int) error {
	if !rt.Config.Defaults.Proxy.Continuity.Enabled || skip {
		return nil
	}

	meta, _, err := rt.Client.Profiles.GetMeta(context.Background(), profileID)
	if err != nil {
		return err
	}
	var current *mlx.Proxy
	if meta != nil && meta.Parameters != nil {
		current = meta.Parameters.Proxy
	}

	pc := rt.Config.Defaults.Proxy.Continuity
	chosen, changed, err := rt.Client.Proxies.EnsureHealthyProxy(
		context.Background(),
		current,
		mlx.EnsureHealthyProfileProxyOptions{
			EnsureHealthyProxyOptions: mlx.EnsureHealthyProxyOptions{
				ThresholdMs:        firstPositive(thresholdOverride, pc.LatencyThresholdMs),
				HardCapMs:          firstPositive(hardCapOverride, pc.LatencyHardCapMs),
				CandidatesPerRound: pc.CandidatesPerRound,
				Checker: mlx.NewHTTPProxyChecker(mlx.HTTPProxyCheckerConfig{
					Targets:          pc.CheckTargets,
					PerTargetTimeout: pc.CheckTimeout.Duration(),
				}),
			},
			PreferSOCKS5: rt.Config.Defaults.Proxy.PreferSOCKS5,
			SaveTraffic:  rt.Config.Defaults.Proxy.SaveTraffic,
		},
	)
	if err != nil {
		return fmt.Errorf("pre-launch proxy check failed: %w", err)
	}
	if changed {
		patch := &mlx.PatchProfileRequest{ProfileID: profileID, Proxy: chosen}
		if _, _, err := rt.Client.Profiles.Patch(context.Background(), patch); err != nil {
			return fmt.Errorf("apply replacement proxy: %w", err)
		}
	}
	return nil
}

func runLauncherStop(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("launcher stop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profileID := fs.String("profile-id", "", "profile id")
	profileName := fs.String("profile-name", "", "profile name")
	folderID := fs.String("folder-id", "", "folder id for profile-name lookup")
	ignoreAlreadyStopped := fs.Bool(
		"ignore-already-stopped",
		false,
		"treat already-stopped errors as success",
	)
	wait := fs.Bool("wait", false, "wait for non-running status")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx launcher stop --profile-id <id> | --profile-name <name> [--folder-id <id>] [--ignore-already-stopped] [--wait]",
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
		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.StopProfileByName(
				context.Background(),
				*profileName,
				mlx.StopProfileByNameOptions{
					FindOptions:          buildFindOptions(rt.Config, *folderID),
					IgnoreAlreadyStopped: *ignoreAlreadyStopped,
					WaitForStopped:       *wait,
					PollOptions:          rt.Config.PollOptions(),
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
		stopResp, _, err := rt.Client.Launcher.Stop(
			context.Background(),
			profile.ID,
		)
		if err != nil && !(*ignoreAlreadyStopped && isAlreadyStoppedError(err)) {
			return err
		}
		if !*wait {
			return emit(rt, map[string]any{
				"profile":       profile,
				"stop_response": stopResp,
			})
		}

		statusResp, err := waitForStoppedStatus(
			context.Background(),
			rt,
			profile.ID,
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx launcher stop-all [--type <cloud|local|quick>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Launcher.StopAll(
			context.Background(),
			mlx.StopAllProfilesOptions{Type: *kind},
		)
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
	case "create-local":
		return runProfileCreate(args[1:], global, true)
	case "create-cloud":
		return runProfileCreate(args[1:], global, false)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx profile list [--search <text>] [--removed] [--limit <n>] [--offset <n>] [--storage-type <all|local|cloud>]",
		)
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
		resp, _, err := rt.Client.Profiles.Search(
			context.Background(),
			&mlx.SearchProfilesRequest{
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
			},
		)
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

func runProfileCreate(args []string, global globalOptions, forceLocal ...bool) error {
	fs := flag.NewFlagSet("profile create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("file", "", "path to CreateProfileRequest JSON")
	templateID := fs.String("template-id", "", "profile template resource id")
	name := fs.String(
		"name",
		"",
		"profile name override for template-based creation",
	)
	folderID := fs.String(
		"folder-id",
		"",
		"folder id override for template-based creation",
	)
	local := newOptionalBoolFlag(
		fs,
		"local",
		"create a local profile from the template",
	)
	managedProxy := fs.Bool(
		"managed-proxy",
		false,
		"generate and attach an MLX managed proxy during template-based creation",
	)
	proxyCountry := fs.String("proxy-country", "", "proxy country code")
	proxyRegion := fs.String("proxy-region", "", "proxy region")
	proxyCity := fs.String("proxy-city", "", "proxy city")
	proxyProtocol := fs.String(
		"proxy-protocol",
		"",
		"proxy protocol: socks5 or http",
	)
	proxySessionType := fs.String(
		"proxy-session-type",
		"",
		"proxy session type: sticky or rotating",
	)
	proxyIPTTL := fs.Int("proxy-ip-ttl", 0, "proxy IPTTL")
	proxyStrict := fs.Bool(
		"proxy-strict",
		false,
		"enable strict mode for managed proxy generation",
	)
	proxySaveTraffic := newOptionalBoolFlag(
		fs,
		"proxy-save-traffic",
		"save traffic in the generated profile proxy",
	)
	country := fs.String(
		"country",
		"",
		"ISO country code; sets language/locale/timezone",
	)
	browser := fs.String("browser", "", "browser type: mimic|stealth")
	osFlag := fs.String("os", "", "os type: windows|macos|linux")
	lang := fs.String(
		"lang",
		"",
		"override browser UI language, e.g. de-DE",
	)
	times := fs.Int("times", 0, "number of profiles to create")
	notes := fs.String("notes", "", "profile notes")
	tagsCSV := fs.String("tags", "", "comma-separated tags")
	start := fs.Bool("start", false, "launch the profile after creation")
	wait := fs.Bool("wait", false, "verify created profile metas")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx profile create --file <request.json> [--wait]\n"+
				"       mlx profile create --template-id <template-id> --name <name> [--folder-id <id>] [--local] [--managed-proxy] [--proxy-country <code>] [--proxy-region <name>] [--proxy-city <name>] [--wait]\n"+
				"       mlx profile create --name <name> [--country <code>] [--browser <type>] [--os <type>] [--lang <locale>] [--folder-id <id>] [--local] [--times <n>] [--notes <text>] [--tags <a,b>] [--managed-proxy] [--proxy-country <code>] [--wait]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*file) != "" && strings.TrimSpace(*templateID) != "" {
		return errors.New("--file and --template-id are mutually exclusive")
	}
	mode := ""
	switch {
	case strings.TrimSpace(*file) != "":
		mode = "file"
	case strings.TrimSpace(*templateID) != "":
		mode = "template"
	case strings.TrimSpace(*name) != "":
		mode = "flags"
	default:
		return errors.New("one of --file, --template-id, or --name is required")
	}

	if *start && *wait {
		return errors.New("--start cannot be combined with --wait; run create without --wait (or start the profile separately)")
	}

	var req mlx.CreateProfileRequest

	return withRuntime(global, func(rt *Runtime) error {
		if mode == "flags" {
			resolvedFolderID, err := resolveFolderID(rt, *folderID)
			if err != nil {
				return err
			}
			effBrowser := firstNonEmpty(
				strings.TrimSpace(*browser),
				rt.Config.Defaults.Profile.BrowserType,
			)
			effOS := firstNonEmpty(
				strings.TrimSpace(*osFlag),
				rt.Config.Defaults.Profile.OSType,
			)
			effLocal := local.ValueOr(false)
			if len(forceLocal) > 0 {
				effLocal = forceLocal[0]
			}
			built, err := buildCreateProfileRequestFromFlags(createFromFlagsInput{
				Name:        *name,
				BrowserType: effBrowser,
				OSType:      effOS,
				Country:     strings.TrimSpace(*country),
				Lang:        *lang,
				FolderID:    resolvedFolderID,
				IsLocal:     effLocal,
				Times:       *times,
				Notes:       *notes,
				Tags:        splitCSV(*tagsCSV),
			})
			if err != nil {
				return err
			}
			req = *built

			if *managedProxy || strings.TrimSpace(*proxyCountry) != "" {
				generated, err := rt.Client.Proxies.GenerateProfileProxy(
					context.Background(),
					&mlx.GenerateProfileProxyRequest{
						GenerateProxyRequest: *buildGenerateProxyRequest(
							rt.Config,
							firstNonEmpty(*proxyCountry, *country),
							*proxyRegion,
							*proxyCity,
							*proxyProtocol,
							*proxySessionType,
							*proxyIPTTL,
							1,
							*proxyStrict,
						),
						PreferSOCKS5: strings.TrimSpace(*proxyProtocol) == "" &&
							rt.Config.Defaults.Proxy.PreferSOCKS5,
						SaveTraffic: proxySaveTraffic.ValueOr(
							rt.Config.Defaults.Proxy.SaveTraffic,
						),
					},
				)
				if err != nil {
					return err
				}
				if req.Parameters == nil {
					req.Parameters = &mlx.ProfileParameters{}
				}
				req.Parameters.Proxy = generated.ProfileProxy
			}
		} else if mode == "file" {
			if err := readJSONFile(*file, &req); err != nil {
				return err
			}
		} else {
			metaResp, _, err := rt.Client.Resources.GetMeta(
				context.Background(),
				*templateID,
			)
			if err != nil {
				return err
			}
			downloadResp, _, err := rt.Client.Resources.Download(
				context.Background(),
				*templateID,
			)
			if err != nil {
				return err
			}
			templateDoc, err := loadProfileTemplate(
				metaResp.Data.MetaInfo,
				downloadResp.Path,
			)
			if err != nil {
				return err
			}

			resolvedFolderID, err := resolveFolderID(rt, *folderID)
			if err != nil {
				return err
			}
			effectiveLocal := local.BoolPtr()
			if len(forceLocal) > 0 {
				effectiveLocal = &forceLocal[0]
			}
			templateReq, err := buildCreateProfileRequestFromTemplate(
				templateDoc,
				*name,
				resolvedFolderID,
				effectiveLocal,
			)
			if err != nil {
				return err
			}
			req = *templateReq

			if *managedProxy {
				generated, err := rt.Client.Proxies.GenerateProfileProxy(
					context.Background(),
					&mlx.GenerateProfileProxyRequest{
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
						PreferSOCKS5: strings.TrimSpace(*proxyProtocol) == "" &&
							rt.Config.Defaults.Proxy.PreferSOCKS5,
						SaveTraffic: proxySaveTraffic.ValueOr(
							rt.Config.Defaults.Proxy.SaveTraffic,
						),
					},
				)
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
			resp, err := rt.Client.Workflows.CreateProfilesAndVerify(
				context.Background(),
				&req,
				mlx.CreateProfilesAndVerifyOptions{
					PollOptions: rt.Config.PollOptions(),
				},
			)
			if err != nil {
				return err
			}
			return emit(rt, resp)
		}

		resp, _, err := rt.Client.Profiles.Create(context.Background(), &req)
		if err != nil {
			return err
		}

		if !*start {
			return emit(rt, resp)
		}
		if len(resp.Data.IDs) == 0 {
			return errors.New("--start: create returned no profile ids to launch")
		}

		startedID := resp.Data.IDs[0]
		startOpts := mlx.StartProfileOptions{
			AutomationType: mlx.AutomationType(rt.Config.Defaults.Launcher.AutomationType),
			Headless:       rt.Config.Defaults.Launcher.Headless,
			StrictMode:     rt.Config.Defaults.Launcher.StrictMode,
		}
		// The fail-closed proxy continuity check runs inside the shared starter
		// before the launch; an unhealthy proxy prevents the profile launch.
		started, err := startProfileWithProxyCheck(
			rt,
			firstNonEmpty(*folderID, req.FolderID),
			startedID,
			startOpts,
			false,
			false,
			0,
			0,
		)
		if err != nil {
			return err
		}

		return emit(rt, map[string]any{
			"create": resp,
			"start":  started.StartResponse,
		})
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx profile clone --id <id> | --name <name> [--times <n>]",
		)
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
		resp, _, err := rt.Client.Profiles.Clone(
			context.Background(),
			&mlx.CloneProfileRequest{
				ProfileID: profile.ID,
				Times:     *times,
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx profile move --ids <id1,id2,...> --dest-folder-id <folder-id>",
		)
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
		resp, _, err := rt.Client.Profiles.Move(
			context.Background(),
			&mlx.MoveProfilesRequest{
				DestinationFolderID: *destFolderID,
				IDs:                 idList,
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx profile delete --ids <id1,id2,...> [--permanently]",
		)
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
		resp, _, err := rt.Client.Profiles.Delete(
			context.Background(),
			&mlx.DeleteProfilesRequest{
				IDs:         idList,
				Permanently: *permanently,
			},
		)
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
		resp, _, err := rt.Client.Profiles.Restore(
			context.Background(),
			&mlx.RestoreProfilesRequest{IDs: idList},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx profile summary --id <id> | --name <name>",
		)
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
		resp, _, err := rt.Client.Profiles.GetSummary(
			context.Background(),
			profile.ID,
		)
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
	profileNameOverride := fs.String(
		"profile-name-override",
		"",
		"archive profile name override",
	)
	stopBeforeExport := newOptionalBoolFlag(
		fs,
		"stop-before-export",
		"stop profile before export",
	)
	ignoreStopNotReady := newOptionalBoolFlag(
		fs,
		"ignore-stop-not-ready",
		"ignore stop errors for not-ready profiles",
	)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx export run --root-dir <dir> (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--folder-name <name>] [--profile-name-override <name>] [--stop-before-export] [--ignore-stop-not-ready]",
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
	if strings.TrimSpace(*rootDir) == "" {
		return errors.New("--root-dir is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		effectiveStopBeforeExport := stopBeforeExport.ValueOr(
			rt.Config.Defaults.Export.StopBeforeExport,
		)
		effectiveIgnoreStopNotReady := ignoreStopNotReady.ValueOr(
			rt.Config.Defaults.Export.IgnoreStopNotReady,
		)
		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.ExportProfileByNameToFolder(
				context.Background(),
				*profileName,
				mlx.ExportProfileByNameToFolderOptions{
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
		if effectiveStopBeforeExport {
			_, _, stopErr := rt.Client.Launcher.Stop(
				context.Background(),
				profile.ID,
			)
			if stopErr != nil &&
				!(effectiveIgnoreStopNotReady && isAlreadyStoppedError(stopErr)) {
				return stopErr
			}
		}
		profileNameForArchive := firstNonEmpty(*profileNameOverride, profile.Name)
		resp, err := rt.Client.Archives.ExportProfileToFolder(
			context.Background(),
			profile.ID,
			mlx.ExportProfileToFolderOptions{
				RootDir:      *rootDir,
				FolderName:   *folderName,
				ProfileName:  profileNameForArchive,
				PollInterval: rt.Config.Poll.InitialInterval.Duration(),
				WaitTimeout:  rt.Config.Poll.Timeout.Duration(),
			},
		)
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
		resp, _, err := rt.Client.Transfers.ExportStatus(
			context.Background(),
			*exportID,
		)
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
		resp, _, err := rt.Client.Transfers.ExportStatuses(
			context.Background(),
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx import run --import-path <archive.zip> [--is-local] [--wait]",
		)
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
			resp, err := rt.Client.Workflows.ImportProfileAndVerify(
				context.Background(),
				req,
				mlx.ImportProfileWorkflowOptions{
					PollOptions: rt.Config.PollOptions(),
				},
			)
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
		resp, _, err := rt.Client.Transfers.ImportStatus(
			context.Background(),
			*importID,
		)
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
		resp, _, err := rt.Client.Transfers.ImportStatuses(
			context.Background(),
		)
		if err != nil {
			return err
		}

		return emit(rt, resp.Data.Statuses)
	})
}
