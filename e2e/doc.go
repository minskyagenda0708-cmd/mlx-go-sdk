//go:build e2e
// +build e2e

// Package e2e contains live end-to-end validation for mlx-go-sdk.
//
// These tests are intentionally separated from the default unit and integration
// flows because they talk to real Multilogin X services and, where applicable,
// a real launcher instance.
//
// Run the E2E suite explicitly with:
//
//	MLX_RUN_E2E=1 go test -tags=e2e ./e2e
//
// Required environment:
//
//   - MLX_TOKEN
//
// Common optional environment overrides:
//
//   - MLX_BASE_URL
//   - MLX_LAUNCHER_URL
//   - MLX_COOKIES_URL
//   - MLX_PROXY_URL
//   - MLX_E2E_FOLDER_ID
//   - MLX_E2E_PROFILE_ID
//   - MLX_E2E_PROFILE_CAP
//
// Additional opt-in spike guards may be required for destructive or rate-limit
// experiments, for example:
//
//   - MLX_RUN_CREATION_LIMIT_SPIKE=1
//   - MLX_RUN_CREATE_50_SPIKE=1
//
// The package keeps both the `e2e` build tag and the runtime
// `MLX_RUN_E2E=1` guard so that ordinary `go test ./...` stays simple and safe,
// while live validation remains an explicit opt-in workflow.
package e2e
