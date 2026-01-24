package declcfg

import (
	"fmt"
	"regexp"
	"time"
)

var versionXYRegex = regexp.MustCompile(`^\d+\.\d+$`)

func ValidateVersionLifecycles(pkgName string, lifecycles []VersionLifecycle) error {
	if len(lifecycles) == 0 {
		return nil
	}

	seenVersions := make(map[string]bool)

	for i, lc := range lifecycles {
		// Validate version format (X.Y)
		if !versionXYRegex.MatchString(lc.Version) {
			return fmt.Errorf("package %q: versionLifecycles[%d]: version %q must be in X.Y format",
				pkgName, i, lc.Version)
		}

		// Check duplicate versions
		if seenVersions[lc.Version] {
			return fmt.Errorf("package %q: duplicate versionLifecycle for version %q",
				pkgName, lc.Version)
		}
		seenVersions[lc.Version] = true

		// Validate compatibility
		if err := validateCompatibility(pkgName, i, lc.Compatibility); err != nil {
			return err
		}

		// Validate phases
		if err := validatePhases(pkgName, i, lc.Phases); err != nil {
			return err
		}
	}
	return nil
}

func validateCompatibility(pkgName string, idx int, compatibility []PlatformCompatibility) error {
	seenPlatforms := make(map[string]bool)

	for j, pc := range compatibility {
		// Platform must be non-empty
		if pc.Platform == "" {
			return fmt.Errorf("package %q: versionLifecycles[%d].compatibility[%d]: platform must not be empty",
				pkgName, idx, j)
		}

		// Check duplicate platforms
		if seenPlatforms[pc.Platform] {
			return fmt.Errorf("package %q: versionLifecycles[%d]: duplicate platform %q",
				pkgName, idx, pc.Platform)
		}
		seenPlatforms[pc.Platform] = true

		// Versions list must not be empty
		if len(pc.Versions) == 0 {
			return fmt.Errorf("package %q: versionLifecycles[%d].compatibility[%d]: versions list must not be empty",
				pkgName, idx, j)
		}

		// Validate version formats (X.Y)
		for k, v := range pc.Versions {
			if !versionXYRegex.MatchString(v) {
				return fmt.Errorf("package %q: versionLifecycles[%d].compatibility[%d].versions[%d]: version %q must be in X.Y format",
					pkgName, idx, j, k, v)
			}
		}
	}
	return nil
}

func validatePhases(pkgName string, idx int, phases []LifecyclePhase) error {
	if len(phases) == 0 {
		return nil
	}

	var prevEndDate time.Time

	for j, phase := range phases {
		// Phase name must be non-empty
		if phase.Name == "" {
			return fmt.Errorf("package %q: versionLifecycles[%d].phases[%d]: name must not be empty",
				pkgName, idx, j)
		}

		// startDate is required and must be valid RFC3339
		if phase.StartDate == "" {
			return fmt.Errorf("package %q: versionLifecycles[%d].phases[%d]: startDate is required",
				pkgName, idx, j)
		}
		startDate, err := time.Parse(time.RFC3339, phase.StartDate)
		if err != nil {
			return fmt.Errorf("package %q: versionLifecycles[%d].phases[%d]: invalid startDate %q: must be RFC3339 format",
				pkgName, idx, j, phase.StartDate)
		}

		// endDate is optional but must be valid RFC3339 if present
		var endDate time.Time
		var hasEndDate bool
		if phase.EndDate != "" {
			hasEndDate = true
			endDate, err = time.Parse(time.RFC3339, phase.EndDate)
			if err != nil {
				return fmt.Errorf("package %q: versionLifecycles[%d].phases[%d]: invalid endDate %q: must be RFC3339 format",
					pkgName, idx, j, phase.EndDate)
			}
			// startDate must be before endDate
			if !startDate.Before(endDate) {
				return fmt.Errorf("package %q: versionLifecycles[%d].phases[%d]: startDate must be before endDate",
					pkgName, idx, j)
			}
		}

		// Phases must be in chronological order
		if j > 0 && !prevEndDate.IsZero() && startDate.Before(prevEndDate) {
			return fmt.Errorf("package %q: versionLifecycles[%d].phases[%d]: phases must be in chronological order",
				pkgName, idx, j)
		}

		if hasEndDate {
			prevEndDate = endDate
		}
	}
	return nil
}
