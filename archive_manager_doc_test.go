package mlx

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultArchiveFolderNameSanitizesWindowsUnsafeCharacters(t *testing.T) {
	name := DefaultArchiveFolderName(`John: Doe/QA`, "profile-1", time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC))
	for _, forbidden := range []string{":", "/", `\\`, "*", "?"} {
		if strings.Contains(name, forbidden) {
			t.Fatalf("folder name must be sanitized, got %s", name)
		}
	}
	if name == "" {
		t.Fatalf("folder name must not be empty")
	}
}
