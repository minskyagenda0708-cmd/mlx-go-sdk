package mlx

import "os"

const (
	// EnvToken is the environment variable that stores the long-lived MultiloginX token.
	EnvToken = "MLX_TOKEN"
)

func tokenFromEnv() string {
	return os.Getenv(EnvToken)
}
