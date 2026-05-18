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
	"strings"

	mlx "github.com/minskyagenda0708-cmd/mlx-go-sdk"
)

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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx folder create --name <name> [--comment <text>]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("--name is required")
	}

	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Folders.Create(
			context.Background(),
			&mlx.CreateFolderRequest{
				Name:    *name,
				Comment: *comment,
			},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx folder update --id <folder-id> --name <name> [--comment <text>]",
		)
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
		resp, _, err := rt.Client.Folders.Update(
			context.Background(),
			&mlx.UpdateFolderRequest{
				FolderID: *id,
				Name:     *name,
				Comment:  *comment,
			},
		)
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
		resp, _, err := rt.Client.Folders.Delete(
			context.Background(),
			&mlx.DeleteFoldersRequest{IDs: idList},
		)
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
		fmt.Fprintln(
			os.Stdout,
			"Usage: mlx template list [--name <text>] [--limit <n>] [--offset <n>] [--trashbin]",
		)
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}

	return withRuntime(global, func(rt *Runtime) error {
		tb := *trashbin
		resp, _, err := rt.Client.Resources.ListProfileTemplates(
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
		metaResp, _, err := rt.Client.Resources.GetMeta(
			context.Background(),
			*id,
		)
		if err != nil {
			return err
		}
		downloadResp, _, err := rt.Client.Resources.Download(
			context.Background(),
			*id,
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
		return emit(rt, map[string]any{
			"meta":     metaResp.Data,
			"path":     downloadResp.Path,
			"template": templateDoc,
		})
	})
}
