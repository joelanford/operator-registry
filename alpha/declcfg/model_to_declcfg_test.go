package declcfg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/operator-framework/operator-registry/alpha/model"
)

func TestConvertFromModel(t *testing.T) {
	type spec struct {
		name      string
		m         model.Model
		expectCfg DeclarativeConfig
	}

	specs := []spec{
		{
			name:      "Success",
			m:         buildTestModel(),
			expectCfg: buildValidDeclarativeConfig(validDeclarativeConfigSpec{IncludeUnrecognized: false, IncludeDeprecations: false}),
		},
		{
			name:      "Success/WithVersionLifecycles",
			m:         buildTestModelWithLifecycles(),
			expectCfg: buildValidDeclarativeConfig(validDeclarativeConfigSpec{IncludeUnrecognized: false, IncludeDeprecations: false, IncludeVersionLifecycles: true}),
		},
	}

	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			s.m.Normalize()
			require.NoError(t, s.m.Validate())
			actual := ConvertFromModel(s.m)

			removeJSONWhitespace(&s.expectCfg)
			removeJSONWhitespace(&actual)

			assert.Equal(t, s.expectCfg, actual)
		})
	}
}

func TestConvertFromModel_VersionLifecycleConversion(t *testing.T) {
	// Create a model with version lifecycles
	techPreviewStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	gaStart := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	eolStart := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	pkg := &model.Package{
		Name:        "test-pkg",
		Description: "test package",
		Channels:    map[string]*model.Channel{},
		VersionLifecycles: []model.VersionLifecycle{
			{
				Version: "1.0",
				Compatibility: []model.PlatformCompatibility{
					{Platform: "OpenShift", Versions: []string{"4.14", "4.15"}},
					{Platform: "Kubernetes", Versions: []string{"1.27", "1.28"}},
				},
				Phases: []model.LifecyclePhase{
					{Name: "tech-preview", StartDate: techPreviewStart, EndDate: &gaStart},
					{Name: "GA", StartDate: gaStart, EndDate: &eolStart},
					{Name: "EOL", StartDate: eolStart, EndDate: nil},
				},
			},
			{
				Version: "2.0",
				Compatibility: []model.PlatformCompatibility{
					{Platform: "OpenShift", Versions: []string{"4.16"}},
				},
				Phases: []model.LifecyclePhase{
					{Name: "GA", StartDate: gaStart, EndDate: nil},
				},
			},
		},
	}

	// Add a channel and bundle so the model is valid
	ch := &model.Channel{
		Package: pkg,
		Name:    "stable",
		Bundles: map[string]*model.Bundle{},
	}
	ch.Bundles["test-pkg.v1.0.0"] = &model.Bundle{
		Package: pkg,
		Channel: ch,
		Name:    "test-pkg.v1.0.0",
	}
	pkg.Channels["stable"] = ch
	pkg.DefaultChannel = ch

	m := model.Model{"test-pkg": pkg}

	// Convert to declcfg
	cfg := ConvertFromModel(m)

	// Verify the conversion
	require.Len(t, cfg.Packages, 1)
	resultPkg := cfg.Packages[0]

	require.Len(t, resultPkg.VersionLifecycles, 2)

	// Check first lifecycle
	lc1 := resultPkg.VersionLifecycles[0]
	assert.Equal(t, "1.0", lc1.Version)
	require.Len(t, lc1.Compatibility, 2)
	assert.Equal(t, "OpenShift", lc1.Compatibility[0].Platform)
	assert.Equal(t, []string{"4.14", "4.15"}, lc1.Compatibility[0].Versions)
	assert.Equal(t, "Kubernetes", lc1.Compatibility[1].Platform)

	require.Len(t, lc1.Phases, 3)
	assert.Equal(t, "tech-preview", lc1.Phases[0].Name)
	assert.Equal(t, "2024-01-15T00:00:00Z", lc1.Phases[0].StartDate)
	assert.Equal(t, "2024-04-01T00:00:00Z", lc1.Phases[0].EndDate)

	assert.Equal(t, "GA", lc1.Phases[1].Name)
	assert.Equal(t, "2024-04-01T00:00:00Z", lc1.Phases[1].StartDate)
	assert.Equal(t, "2025-12-31T23:59:59Z", lc1.Phases[1].EndDate)

	assert.Equal(t, "EOL", lc1.Phases[2].Name)
	assert.Equal(t, "2025-12-31T23:59:59Z", lc1.Phases[2].StartDate)
	assert.Empty(t, lc1.Phases[2].EndDate, "EOL phase should have empty end date")

	// Check second lifecycle
	lc2 := resultPkg.VersionLifecycles[1]
	assert.Equal(t, "2.0", lc2.Version)
	require.Len(t, lc2.Phases, 1)
	assert.Empty(t, lc2.Phases[0].EndDate, "open-ended phase should have empty end date")
}

func TestConvertFromModel_EmptyVersionLifecycles(t *testing.T) {
	// Model without version lifecycles should result in nil/empty lifecycles
	m := buildTestModel()

	cfg := ConvertFromModel(m)

	for _, pkg := range cfg.Packages {
		assert.Nil(t, pkg.VersionLifecycles, "packages without lifecycles should have nil VersionLifecycles")
	}
}

func TestConvertFromModel_VersionLifecycleRoundTrip(t *testing.T) {
	// Start with a DeclarativeConfig with version lifecycles
	original := buildValidDeclarativeConfig(validDeclarativeConfigSpec{
		IncludeUnrecognized:      false,
		IncludeDeprecations:      false,
		IncludeVersionLifecycles: true,
	})

	// Convert to model
	m, err := ConvertToModel(original)
	require.NoError(t, err)

	// Convert back to declcfg
	roundTripped := ConvertFromModel(m)

	// Remove JSON whitespace for comparison
	removeJSONWhitespace(&original)
	removeJSONWhitespace(&roundTripped)

	// Compare packages (including version lifecycles)
	assert.Equal(t, original.Packages, roundTripped.Packages)
}
