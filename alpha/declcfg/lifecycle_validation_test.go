package declcfg

import (
	"testing"
)

func TestValidateVersionLifecycles(t *testing.T) {
	tests := []struct {
		name       string
		pkgName    string
		lifecycles []VersionLifecycle
		wantErr    bool
		errContain string
	}{
		{
			name:       "nil lifecycles is valid",
			pkgName:    "test-pkg",
			lifecycles: nil,
			wantErr:    false,
		},
		{
			name:       "empty lifecycles is valid",
			pkgName:    "test-pkg",
			lifecycles: []VersionLifecycle{},
			wantErr:    false,
		},
		{
			name:    "valid lifecycle with all fields",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Compatibility: []PlatformCompatibility{
						{
							Platform: "OpenShift",
							Versions: []string{"4.14", "4.15"},
						},
					},
					Phases: []LifecyclePhase{
						{
							Name:      "GA",
							StartDate: "2024-01-01T00:00:00Z",
							EndDate:   "2024-12-31T23:59:59Z",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "valid lifecycle with multiple versions",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: "1.0"},
				{Version: "1.1"},
				{Version: "2.0"},
			},
			wantErr: false,
		},
		{
			name:    "invalid version format - single number",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: "1"},
			},
			wantErr:    true,
			errContain: "must be in X.Y format",
		},
		{
			name:    "invalid version format - three parts",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: "1.2.3"},
			},
			wantErr:    true,
			errContain: "must be in X.Y format",
		},
		{
			name:    "invalid version format - empty",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: ""},
			},
			wantErr:    true,
			errContain: "must be in X.Y format",
		},
		{
			name:    "invalid version format - non-numeric",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: "v1.2"},
			},
			wantErr:    true,
			errContain: "must be in X.Y format",
		},
		{
			name:    "duplicate version",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: "1.2"},
				{Version: "1.2"},
			},
			wantErr:    true,
			errContain: "duplicate versionLifecycle for version",
		},
		{
			name:    "empty platform name",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Compatibility: []PlatformCompatibility{
						{Platform: "", Versions: []string{"4.14"}},
					},
				},
			},
			wantErr:    true,
			errContain: "platform must not be empty",
		},
		{
			name:    "duplicate platform",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Compatibility: []PlatformCompatibility{
						{Platform: "OpenShift", Versions: []string{"4.14"}},
						{Platform: "OpenShift", Versions: []string{"4.15"}},
					},
				},
			},
			wantErr:    true,
			errContain: "duplicate platform",
		},
		{
			name:    "empty versions list",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Compatibility: []PlatformCompatibility{
						{Platform: "OpenShift", Versions: []string{}},
					},
				},
			},
			wantErr:    true,
			errContain: "versions list must not be empty",
		},
		{
			name:    "invalid platform version format",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Compatibility: []PlatformCompatibility{
						{Platform: "OpenShift", Versions: []string{"4.14.1"}},
					},
				},
			},
			wantErr:    true,
			errContain: "must be in X.Y format",
		},
		{
			name:    "empty phase name",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "", StartDate: "2024-01-01T00:00:00Z"},
					},
				},
			},
			wantErr:    true,
			errContain: "name must not be empty",
		},
		{
			name:    "missing startDate",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "GA", StartDate: ""},
					},
				},
			},
			wantErr:    true,
			errContain: "startDate is required",
		},
		{
			name:    "invalid startDate format",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "GA", StartDate: "2024-01-01"},
					},
				},
			},
			wantErr:    true,
			errContain: "must be RFC3339 format",
		},
		{
			name:    "invalid endDate format",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "GA", StartDate: "2024-01-01T00:00:00Z", EndDate: "2024-12-31"},
					},
				},
			},
			wantErr:    true,
			errContain: "must be RFC3339 format",
		},
		{
			name:    "startDate after endDate",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "GA", StartDate: "2024-12-31T00:00:00Z", EndDate: "2024-01-01T00:00:00Z"},
					},
				},
			},
			wantErr:    true,
			errContain: "startDate must be before endDate",
		},
		{
			name:    "phases not in chronological order",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "GA", StartDate: "2024-06-01T00:00:00Z", EndDate: "2024-12-31T00:00:00Z"},
						{Name: "tech-preview", StartDate: "2024-01-01T00:00:00Z", EndDate: "2024-03-01T00:00:00Z"},
					},
				},
			},
			wantErr:    true,
			errContain: "phases must be in chronological order",
		},
		{
			name:    "valid phases in chronological order",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "tech-preview", StartDate: "2024-01-01T00:00:00Z", EndDate: "2024-03-01T00:00:00Z"},
						{Name: "GA", StartDate: "2024-03-01T00:00:00Z", EndDate: "2025-06-01T00:00:00Z"},
						{Name: "EOL", StartDate: "2025-06-01T00:00:00Z"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "phase without endDate is valid (terminal phase)",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases: []LifecyclePhase{
						{Name: "EOL", StartDate: "2025-06-01T00:00:00Z"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "lifecycle with only version is valid",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{Version: "1.2"},
			},
			wantErr: false,
		},
		{
			name:    "lifecycle with empty compatibility is valid",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version:       "1.2",
					Compatibility: []PlatformCompatibility{},
				},
			},
			wantErr: false,
		},
		{
			name:    "lifecycle with empty phases is valid",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Phases:  []LifecyclePhase{},
				},
			},
			wantErr: false,
		},
		{
			name:    "multiple platforms",
			pkgName: "test-pkg",
			lifecycles: []VersionLifecycle{
				{
					Version: "1.2",
					Compatibility: []PlatformCompatibility{
						{Platform: "OpenShift", Versions: []string{"4.14", "4.15", "4.16"}},
						{Platform: "Kubernetes", Versions: []string{"1.27", "1.28", "1.29"}},
						{Platform: "OKD", Versions: []string{"4.14"}},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersionLifecycles(tt.pkgName, tt.lifecycles)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersionLifecycles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if err == nil || !containsString(err.Error(), tt.errContain) {
					t.Errorf("ValidateVersionLifecycles() error = %v, expected to contain %q", err, tt.errContain)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
