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

func TestBuildGenerateProxyRequestUsesConfiguredGeoDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.Proxy.Country = "us"
	cfg.Defaults.Proxy.Region = "new_jersey"
	cfg.Defaults.Proxy.City = "east_brunswick"

	req := buildGenerateProxyRequest(cfg, "", "", "", "", "", 0, 0, false)
	if req.Country != "us" {
		t.Fatalf("expected default country, got %q", req.Country)
	}
	if req.Region != "new_jersey" {
		t.Fatalf("expected default region, got %q", req.Region)
	}
	if req.City != "east_brunswick" {
		t.Fatalf("expected default city, got %q", req.City)
	}
}

func TestExecuteHelpTag(t *testing.T) {
	if err := Execute([]string{"help", "tag"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestExecuteHelpLauncherQuickStart(t *testing.T) {
	if err := Execute([]string{"help", "launcher"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestExecuteHelpProxyValidate(t *testing.T) {
	if err := Execute([]string{"help", "proxy"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestExecuteHelpProfileCreateLocal(t *testing.T) {
	if err := Execute([]string{"help", "profile"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}
