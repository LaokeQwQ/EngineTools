package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name   string
		remote string
		local  string
		want   bool
	}{
		{"newer patch", "v1.9.1", "1.9.0", true},
		{"same version", "v1.9.0", "1.9.0", false},
		{"older remote", "v1.8.0", "1.9.0", false},
		{"stable after preview", "v1.9.0", "1.9.0 Preview", true},
		{"preview after stable", "v1.9.0-preview", "1.9.0", false},
		{"same preview", "v1.9.0-preview", "1.9.0 Preview", false},
		{"newer preview", "v1.10.0-preview", "1.9.0 Preview", true},
	}
	for _, c := range cases {
		if got := isNewer(c.remote, c.local); got != c.want {
			t.Errorf("%s: isNewer(%q, %q) = %v, want %v", c.name, c.remote, c.local, got, c.want)
		}
	}
}
