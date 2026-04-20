package mlx

import "testing"

func TestNewFromEnvUsesURLOverrides(t *testing.T) {
	t.Setenv(EnvToken, "test-token")
	t.Setenv(EnvBaseURL, "https://api.example.test")
	t.Setenv(EnvLauncherURL, "https://launcher.example.test:45001")
	t.Setenv(EnvCookiesURL, "https://cookies.example.test")

	client, err := NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv returned error: %v", err)
	}

	if got := client.baseURL.String(); got != "https://api.example.test/" {
		t.Fatalf("unexpected base url: %s", got)
	}
	if got := client.launcherURL.String(); got != "https://launcher.example.test:45001/" {
		t.Fatalf("unexpected launcher url: %s", got)
	}
	if got := client.cookiesURL.String(); got != "https://cookies.example.test/" {
		t.Fatalf("unexpected cookies url: %s", got)
	}
}
