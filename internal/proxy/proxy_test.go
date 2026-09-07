package proxy

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"my-app.dev": "my-app", "my-app": "my-app", "dev.dev": "dev"} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
