package registry

import "testing"

func TestClassifyWindowsVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		major       uint32
		minor       uint32
		build       uint32
		productType byte
		want        string
	}{
		{
			name:        "Windows 10 22H2 workstation",
			major:       10,
			minor:       0,
			build:       19045,
			productType: verNTWorkstation,
			want:        "Windows 10",
		},
		{
			name:        "last build before Windows 11",
			major:       10,
			minor:       0,
			build:       21999,
			productType: verNTWorkstation,
			want:        "Windows 10",
		},
		{
			name:        "first Windows 11 build",
			major:       10,
			minor:       0,
			build:       22000,
			productType: verNTWorkstation,
			want:        "Windows 11",
		},
		{
			name:        "Windows 11 23H2 workstation",
			major:       10,
			minor:       0,
			build:       22631,
			productType: verNTWorkstation,
			want:        "Windows 11",
		},
		{
			name:        "Windows Server 2022",
			major:       10,
			minor:       0,
			build:       20348,
			productType: verNTServer,
			want:        "Windows Server",
		},
		{
			name:        "Windows Server 2025 shares a Windows 11 build",
			major:       10,
			minor:       0,
			build:       26100,
			productType: verNTServer,
			want:        "Windows Server",
		},
		{
			name:        "unknown older Windows version keeps numeric identity",
			major:       6,
			minor:       3,
			build:       9600,
			productType: verNTWorkstation,
			want:        "Windows 6.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := classifyWindowsVersion(tt.major, tt.minor, tt.build, tt.productType)
			if got != tt.want {
				t.Fatalf("classifyWindowsVersion(%d, %d, %d, %d) = %q, want %q",
					tt.major, tt.minor, tt.build, tt.productType, got, tt.want)
			}
		})
	}
}

func TestFormatWindowsVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		family         string
		displayVersion string
		build          uint32
		ubr            uint64
		want           string
	}{
		{
			name:           "feature version and update build revision",
			family:         "Windows 11",
			displayVersion: "24H2",
			build:          26100,
			ubr:            4652,
			want:           "Windows 11 24H2 (Build 26100.4652)",
		},
		{
			name:   "build without optional registry metadata",
			family: "Windows 10",
			build:  19045,
			want:   "Windows 10 (Build 19045)",
		},
		{
			name:           "unknown kernel version keeps available feature version",
			family:         "Unknown",
			displayVersion: "24H2",
			want:           "Unknown 24H2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatWindowsVersion(tt.family, tt.displayVersion, tt.build, tt.ubr)
			if got != tt.want {
				t.Fatalf("formatWindowsVersion(%q, %q, %d, %d) = %q, want %q",
					tt.family, tt.displayVersion, tt.build, tt.ubr, got, tt.want)
			}
		})
	}
}
