package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mlx "github.com/bath0ry/mlx-go-sdk"
)

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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension list [--name <text>] [--limit <n>] [--offset <n>] [--trashbin]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		tb := *trashbin
		resp, _, err := rt.Client.Resources.ListExtensions(
			context.Background(),
			&mlx.ListResourceMetasOptions{
				ObjectName: *name,
				Limit:      *limit,
				Offset:     *offset,
				Trashbin:   &tb,
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension upload --path <zip> [--storage-type <cloud|local>]",
		)
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
		resp, _, err := rt.Client.Resources.UploadExtension(
			context.Background(),
			&mlx.UploadExtensionRequest{
				ObjectPath:  *path,
				StorageType: effectiveStorage,
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension create-url --url <download-url> [--browser-type <browser>] [--storage-type <cloud|local>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*rawURL) == "" {
		return errors.New("--url is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.CreateExtensionFromURL(
			context.Background(),
			&mlx.CreateExtensionFromURLRequest{
				URL: *rawURL,
				BrowserType: firstNonEmpty(
					*browserType,
					rt.Config.Defaults.Extension.BrowserType,
				),
				StorageType: firstNonEmpty(
					*storageType,
					rt.Config.Defaults.Extension.StorageType,
				),
			},
		)
		if err != nil {
			return err
		}

		return emit(rt, resp)
	})
}

func runExtensionCreateWebStore(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("extension create-webstore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	extensionID := fs.String(
		"extension-id",
		"",
		"Chrome Web Store extension id",
	)
	browserType := fs.String("browser-type", "", "browser type")
	storageType := fs.String("storage-type", "", "cloud|local")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension create-webstore --extension-id <id> [--browser-type <browser>] [--storage-type <cloud|local>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*extensionID) == "" {
		return errors.New("--extension-id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.CreateExtensionFromChromeWebStore(
			context.Background(),
			&mlx.CreateChromeWebStoreExtensionRequest{
				ExtensionID: *extensionID,
				BrowserType: firstNonEmpty(
					*browserType,
					rt.Config.Defaults.Extension.BrowserType,
				),
				StorageType: firstNonEmpty(
					*storageType,
					rt.Config.Defaults.Extension.StorageType,
				),
			},
		)
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
	requireProfileUsageRead := newOptionalBoolFlag(
		fs,
		"require-profile-usage-read",
		"require profile usage read verification",
	)
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension enable --id <resource-id> (--profile-id <id> | --profile-name <name>) [--folder-id <id>] [--require-profile-usage-read]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*resourceID) == "" {
		return errors.New("--id is required")
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
		profileUsageReadRequired := requireProfileUsageRead.ValueOr(
			rt.Config.Defaults.Extension.RequireProfileUsageRead,
		)
		if strings.TrimSpace(*profileName) != "" {
			resp, err := rt.Client.Workflows.EnableExtensionForProfileByName(
				context.Background(),
				*profileName,
				*resourceID,
				mlx.EnableExtensionForProfileByNameOptions{
					FindOptions:             buildFindOptions(rt.Config, *folderID),
					PollOptions:             rt.Config.PollOptions(),
					RequireProfileUsageRead: profileUsageReadRequired,
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
		enableResp, _, err := rt.Client.Resources.EnableExtensionForProfiles(
			context.Background(),
			*resourceID,
			&mlx.SetResourceProfilesRequest{
				ProfileIDs: []string{profile.ID},
			},
		)
		if err != nil {
			return err
		}
		usages, err := waitForObjectUsageAttached(
			context.Background(),
			rt,
			*resourceID,
			profile.ID,
		)
		if err != nil {
			return err
		}
		profileUsages, _, profileUsageErr := rt.Client.Resources.ProfileExtensionUsages(
			context.Background(),
			profile.ID,
		)
		if profileUsageErr != nil && profileUsageReadRequired {
			return profileUsageErr
		}

		return emit(rt, map[string]any{
			"profile":         profile,
			"enable_response": enableResp,
			"object_usages":   usages,
			"profile_usages":  profileUsages,
			"profile_error":   errorString(profileUsageErr),
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension disable --id <resource-id> (--profile-id <id> | --profile-name <name>) [--folder-id <id>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*resourceID) == "" {
		return errors.New("--id is required")
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
		resp, _, err := rt.Client.Resources.DisableExtensionForProfiles(
			context.Background(),
			*resourceID,
			&mlx.SetResourceProfilesRequest{
				ProfileIDs: []string{profile.ID},
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension usages --id <resource-id> | (--profile-id <id> | --profile-name <name>)",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*resourceID) == "" &&
		strings.TrimSpace(*profileID) == "" &&
		strings.TrimSpace(*profileName) == "" {
		return errors.New("one of --id, --profile-id, or --profile-name is required")
	}
	if strings.TrimSpace(*resourceID) != "" &&
		(strings.TrimSpace(*profileID) != "" ||
			strings.TrimSpace(*profileName) != "") {
		return errors.New("--id cannot be combined with --profile-id or --profile-name")
	}
	if strings.TrimSpace(*profileID) != "" &&
		strings.TrimSpace(*profileName) != "" {
		return errors.New("--profile-id and --profile-name are mutually exclusive")
	}

	return withRuntime(global, func(rt *Runtime) error {
		if strings.TrimSpace(*resourceID) != "" {
			resp, _, err := rt.Client.Resources.ObjectProfileUsages(
				context.Background(),
				*resourceID,
			)
			if err != nil {
				return err
			}

			return emit(rt, resp.Data)
		}

		profile, err := resolveProfile(rt, *profileID, *profileName, *folderID)
		if err != nil {
			return err
		}
		resp, _, err := rt.Client.Resources.ProfileExtensionUsages(
			context.Background(),
			profile.ID,
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx extension delete --id <resource-id> [--permanently]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Resources.Delete(
			context.Background(),
			*id,
			*permanently,
		)
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
