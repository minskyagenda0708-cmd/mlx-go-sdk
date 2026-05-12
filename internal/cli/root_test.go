package cli

import (
	"slices"
	"testing"
)

func TestParseGlobalAllowsOutputAfterCommand(t *testing.T) {
	opts, rest, err := parseGlobal([]string{"launcher", "health", "--output", "json"})
	if err != nil {
		t.Fatalf("parseGlobal returned error: %v", err)
	}

	if opts.Output != "json" {
		t.Fatalf("expected output override %q, got %q", "json", opts.Output)
	}

	wantRest := []string{"launcher", "health"}
	if !slices.Equal(rest, wantRest) {
		t.Fatalf("unexpected remaining args: got %#v want %#v", rest, wantRest)
	}
}

func TestParseGlobalAllowsConfigAfterCommand(t *testing.T) {
	opts, rest, err := parseGlobal([]string{"folder", "list", "--config", "configs/cli.json"})
	if err != nil {
		t.Fatalf("parseGlobal returned error: %v", err)
	}

	if opts.ConfigPath != "configs/cli.json" {
		t.Fatalf("expected config path %q, got %q", "configs/cli.json", opts.ConfigPath)
	}

	wantRest := []string{"folder", "list"}
	if !slices.Equal(rest, wantRest) {
		t.Fatalf("unexpected remaining args: got %#v want %#v", rest, wantRest)
	}
}

func TestParseGlobalAllowsMixedGlobalFlagPlacement(t *testing.T) {
	opts, rest, err := parseGlobal([]string{"--output", "yaml", "proxy", "usage", "--config", "cli.json"})
	if err != nil {
		t.Fatalf("parseGlobal returned error: %v", err)
	}

	if opts.Output != "yaml" {
		t.Fatalf("expected output override %q, got %q", "yaml", opts.Output)
	}
	if opts.ConfigPath != "cli.json" {
		t.Fatalf("expected config path %q, got %q", "cli.json", opts.ConfigPath)
	}

	wantRest := []string{"proxy", "usage"}
	if !slices.Equal(rest, wantRest) {
		t.Fatalf("unexpected remaining args: got %#v want %#v", rest, wantRest)
	}
}

func TestParseGlobalPreservesSubcommandFlagsWhileRemovingTrailingGlobals(t *testing.T) {
	args := []string{
		"profile", "list",
		"--limit", "5",
		"--output", "json",
		"--search", "Demo",
		"--config", "cli.json",
	}

	opts, rest, err := parseGlobal(args)
	if err != nil {
		t.Fatalf("parseGlobal returned error: %v", err)
	}

	if opts.Output != "json" {
		t.Fatalf("expected output override %q, got %q", "json", opts.Output)
	}
	if opts.ConfigPath != "cli.json" {
		t.Fatalf("expected config path %q, got %q", "cli.json", opts.ConfigPath)
	}

	wantRest := []string{"profile", "list", "--limit", "5", "--search", "Demo"}
	if !slices.Equal(rest, wantRest) {
		t.Fatalf("unexpected remaining args: got %#v want %#v", rest, wantRest)
	}
}

func TestParseGlobalRejectsTrailingGlobalFlagWithoutValue(t *testing.T) {
	_, _, err := parseGlobal([]string{"launcher", "health", "--output"})
	if err == nil {
		t.Fatal("expected parseGlobal to fail for missing --output value")
	}
}

func TestParseGlobalPreservesSubcommandHelpFlag(t *testing.T) {
	opts, rest, err := parseGlobal([]string{"launcher", "--help"})
	if err != nil {
		t.Fatalf("parseGlobal returned error: %v", err)
	}

	if opts.Help {
		t.Fatal("expected trailing --help to remain available for subcommand parsing")
	}

	wantRest := []string{"launcher", "--help"}
	if !slices.Equal(rest, wantRest) {
		t.Fatalf("unexpected remaining args: got %#v want %#v", rest, wantRest)
	}
}
