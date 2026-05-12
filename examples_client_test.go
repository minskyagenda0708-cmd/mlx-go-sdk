package mlx_test

import (
	"fmt"
	"os"
	"strings"
	"time"

	mlx "mlx-go-sdk"
)

func ExampleNewFromEnv_productionClient() {
	_ = os.Setenv(mlx.EnvToken, "test-token")
	_ = os.Setenv(mlx.EnvBaseURL, "https://api.example.test")
	_ = os.Setenv(mlx.EnvLauncherURL, "https://launcher.example.test:45001")
	_ = os.Setenv(mlx.EnvCookiesURL, "https://cookies.example.test")
	_ = os.Setenv(mlx.EnvProxyURL, "https://proxy.example.test")
	defer os.Unsetenv(mlx.EnvToken)
	defer os.Unsetenv(mlx.EnvBaseURL)
	defer os.Unsetenv(mlx.EnvLauncherURL)
	defer os.Unsetenv(mlx.EnvCookiesURL)
	defer os.Unsetenv(mlx.EnvProxyURL)

	client, err := mlx.NewFromEnv(
		mlx.WithTimeout(30*time.Second),
		mlx.WithRetry(mlx.RetryOptions{
			MaxAttempts:     4,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     2 * time.Second,
			Multiplier:      2,
			Jitter:          0,
		}),
		mlx.WithUserAgent("acme-mlx-cli/1.0"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(client != nil)
	// Output:
	// true
}

func ExampleDefaultArchiveFolderName() {
	name := mlx.DefaultArchiveFolderName(`John: Doe/QA`, "profile-1", time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC))
	fmt.Println(strings.Contains(name, ":"), strings.Contains(name, "/"))
	// Output:
	// false false
}
