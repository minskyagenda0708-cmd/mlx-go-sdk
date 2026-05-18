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

func runTag(args []string, global globalOptions) error {
	if len(args) == 0 {
		printTagHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "create":
		return runTagCreate(args[1:], global)
	case "update":
		return runTagUpdate(args[1:], global)
	case "remove":
		return runTagRemove(args[1:], global)
	case "assign":
		return runTagAssign(args[1:], global)
	case "search":
		return runTagSearch(args[1:], global)
	default:
		printTagHelp(os.Stdout)
		return fmt.Errorf("unknown tag subcommand %q", args[0])
	}
}

func runTagCreate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("tag create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "tag name")
	color := fs.String("color", "gray", "tag color")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx tag create --name NAME [--color COLOR]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("--name is required")
	}
	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Tags.Create(context.Background(), &mlx.CreateTagsRequest{
			Tags: []mlx.CreateTagItem{{Name: *name, Color: *color}},
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runTagUpdate(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("tag update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("id", "", "tag id")
	name := fs.String("name", "", "new tag name")
	color := fs.String("color", "", "new tag color")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx tag update --id ID [--name NAME] [--color COLOR]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return errors.New("--id is required")
	}
	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Tags.Update(context.Background(), &mlx.UpdateTagsRequest{
			Tags: []mlx.UpdateTagItem{{ID: *id, Name: *name, Color: *color}},
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runTagRemove(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("tag remove", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ids := fs.String("ids", "", "comma-separated tag ids")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx tag remove --ids id1,id2,...")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*ids) == "" {
		return errors.New("--ids is required")
	}
	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Tags.Remove(context.Background(), &mlx.RemoveTagsRequest{
			IDs: strings.Split(*ids, ","),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runTagAssign(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("tag assign", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	tagIDs := fs.String("tag-ids", "", "comma-separated tag ids")
	profileIDs := fs.String("profile-ids", "", "comma-separated profile ids")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx tag assign --tag-ids id1,id2,... --profile-ids pid1,pid2,...")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*tagIDs) == "" {
		return errors.New("--tag-ids is required")
	}
	if strings.TrimSpace(*profileIDs) == "" {
		return errors.New("--profile-ids is required")
	}
	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Tags.AssignToProfiles(context.Background(), &mlx.AssignTagsRequest{
			TagIDs:     strings.Split(*tagIDs, ","),
			ProfileIDs: strings.Split(*profileIDs, ","),
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}

func runTagSearch(args []string, global globalOptions) error {
	fs := flag.NewFlagSet("tag search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	searchText := fs.String("search", "", "search text")
	limit := fs.Int("limit", 100, "page size")
	offset := fs.Int("offset", 0, "page offset")
	orderBy := fs.String("order-by", "", "order by field")
	sort := fs.String("sort", "", "asc|desc")
	help := fs.Bool("help", false, "show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Fprintln(os.Stdout, "Usage: mlx tag search [--search TEXT] [--limit N] [--offset N] [--order-by FIELD] [--sort asc|desc]")
		return nil
	}
	if err := requireNoExtraArgs(fs.Args()); err != nil {
		return err
	}
	return withRuntime(global, func(rt *Runtime) error {
		resp, _, err := rt.Client.Tags.Search(context.Background(), &mlx.SearchTagsRequest{
			SearchText: *searchText,
			Limit:      *limit,
			Offset:     *offset,
			OrderBy:    *orderBy,
			Sort:       *sort,
		})
		if err != nil {
			return err
		}
		return emit(rt, resp)
	})
}
