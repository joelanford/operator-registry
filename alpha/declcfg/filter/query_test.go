package filter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
)

func TestNewQueryParser(t *testing.T) {
	parser := NewQueryParser()
	assert.NotNil(t, parser)
}

func TestQueryParser_Parse_Simple(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name      string
		query     string
		wantError bool
	}{
		{
			name:      "empty query",
			query:     "",
			wantError: false,
		},
		{
			name:      "simple any query",
			query:     "maturity:any=stable",
			wantError: false,
		},
		{
			name:      "simple all query",
			query:     "keywords:all=database,sql",
			wantError: false,
		},
		{
			name:      "case insensitive operator",
			query:     "maturity:ANY=stable",
			wantError: false,
		},
		{
			name:      "invalid operator",
			query:     "maturity:invalid=stable",
			wantError: true,
		},
		{
			name:      "missing field",
			query:     ":any=stable",
			wantError: true,
		},
		{
			name:      "missing operator",
			query:     "maturity=stable",
			wantError: true,
		},
		{
			name:      "missing values",
			query:     "maturity:any=",
			wantError: true,
		},
		{
			name:      "multiple values",
			query:     "maturity:any=stable,alpha,beta",
			wantError: false,
		},
		{
			name:      "values with spaces",
			query:     "maturity:any=stable, alpha, beta",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parser.Parse(tt.query)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, filter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, filter)
			}
		})
	}
}

func TestQueryParser_Parse_Complex(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		name      string
		query     string
		wantError bool
	}{
		{
			name:      "explicit AND",
			query:     "maturity:any=stable AND keywords:all=database,sql",
			wantError: false,
		},
		{
			name:      "explicit OR",
			query:     "maturity:any=stable OR keywords:any=database",
			wantError: false,
		},
		{
			name:      "implicit AND",
			query:     "maturity:any=stable keywords:all=database,sql",
			wantError: false,
		},
		{
			name:      "case insensitive AND",
			query:     "maturity:any=stable and keywords:all=database,sql",
			wantError: false,
		},
		{
			name:      "case insensitive OR",
			query:     "maturity:any=stable or keywords:any=database",
			wantError: false,
		},
		{
			name:      "multiple AND",
			query:     "maturity:any=stable AND keywords:all=database,sql AND features:any=backup",
			wantError: false,
		},
		{
			name:      "multiple OR",
			query:     "maturity:any=stable OR keywords:any=database OR features:any=backup",
			wantError: false,
		},
		{
			name:      "mixed AND/OR",
			query:     "maturity:any=stable AND keywords:all=database OR features:any=backup",
			wantError: false,
		},
		{
			name:      "parentheses",
			query:     "maturity:any=stable AND (keywords:all=database OR features:any=backup)",
			wantError: false,
		},
		{
			name:      "nested parentheses",
			query:     "maturity:any=stable AND (keywords:all=database OR (features:any=backup AND features:any=monitoring))",
			wantError: false,
		},
		{
			name:      "mismatched parentheses",
			query:     "maturity:any=stable AND (keywords:all=database",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parser.Parse(tt.query)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, filter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, filter)
			}
		})
	}
}

func TestQueryParser_MustParse(t *testing.T) {
	parser := NewQueryParser()

	// Should not panic on valid query
	filter := parser.MustParse("maturity:any=stable")
	assert.NotNil(t, filter)

	// Should panic on invalid query
	assert.Panics(t, func() {
		parser.MustParse("invalid:query")
	})
}

func TestQueryParser_ParseAndMatch(t *testing.T) {
	parser := NewQueryParser()

	// Create test metas with search metadata
	stableMeta := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "storage", "sql"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true, "monitoring": false}},
	})

	alphaMeta := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "alpha"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"ssl": true, "compression": false}},
	})

	tests := []struct {
		name        string
		query       string
		meta        declcfg.Meta
		expectMatch bool
	}{
		{
			name:        "simple any - matches",
			query:       "maturity:any=stable",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "simple any - no match",
			query:       "maturity:any=stable",
			meta:        alphaMeta,
			expectMatch: false,
		},
		{
			name:        "simple all - matches",
			query:       "keywords:all=database,sql",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "simple all - no match",
			query:       "keywords:all=database,sql",
			meta:        alphaMeta,
			expectMatch: false,
		},
		{
			name:        "multiple values any - matches",
			query:       "maturity:any=stable,alpha,beta",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "multiple values any - no match",
			query:       "maturity:any=beta,gamma",
			meta:        stableMeta,
			expectMatch: false,
		},
		{
			name:        "AND - both match",
			query:       "maturity:any=stable AND keywords:any=database",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "AND - first matches, second doesn't",
			query:       "maturity:any=stable AND keywords:any=web",
			meta:        stableMeta,
			expectMatch: false,
		},
		{
			name:        "OR - first matches",
			query:       "maturity:any=stable OR keywords:any=web",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "OR - second matches",
			query:       "maturity:any=beta OR keywords:any=database",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "OR - neither matches",
			query:       "maturity:any=beta OR keywords:any=web",
			meta:        stableMeta,
			expectMatch: false,
		},
		{
			name:        "implicit AND",
			query:       "maturity:any=stable keywords:any=database",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "parentheses - OR inside AND",
			query:       "maturity:any=stable AND (keywords:any=database OR features:any=ssl)",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "parentheses - AND inside OR",
			query:       "maturity:any=beta OR (keywords:any=database AND features:any=backup)",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "complex nested - matches",
			query:       "maturity:any=stable AND (keywords:all=database,sql OR features:any=backup)",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "features map - matches",
			query:       "features:any=backup",
			meta:        stableMeta,
			expectMatch: true,
		},
		{
			name:        "features map - no match (false value)",
			query:       "features:any=monitoring",
			meta:        stableMeta,
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parser.Parse(tt.query)
			require.NoError(t, err)

			matches, err := filter.MatchMeta(tt.meta)
			require.NoError(t, err)

			assert.Equal(t, tt.expectMatch, matches)
		})
	}
}

func TestQueryParser_EdgeCases(t *testing.T) {
	parser := NewQueryParser()

	// Test empty query returns a filter that matches all
	emptyFilter, err := parser.Parse("")
	require.NoError(t, err)

	meta := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
	})

	matches, err := emptyFilter.MatchMeta(meta)
	require.NoError(t, err)
	assert.True(t, matches)

	// Test query with spaces
	spaceFilter, err := parser.Parse("  maturity:any=stable  ")
	require.NoError(t, err)

	matches, err = spaceFilter.MatchMeta(meta)
	require.NoError(t, err)
	assert.True(t, matches)
}

func TestQueryParser_WhitespaceHandling(t *testing.T) {
	parser := NewQueryParser()

	meta := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
	})

	// Test various whitespace scenarios
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "single line",
			query: "maturity:any=stable AND keywords:all=database,sql",
		},
		{
			name: "with newlines",
			query: `maturity:any=stable AND
			        keywords:all=database,sql`,
		},
		{
			name:  "with tabs and newlines",
			query: "maturity:any=stable\tAND\n\t\tkeywords:all=database,sql",
		},
		{
			name: "complex with newlines and indentation",
			query: `maturity:any=stable AND
			        (keywords:all=database,sql OR
			         keywords:any=storage)`,
		},
		{
			name:  "extra whitespace everywhere",
			query: "  maturity:any=stable   AND   keywords:all=database,sql  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parser.Parse(tt.query)
			require.NoError(t, err, "Failed to parse query: %s", tt.query)

			matches, err := filter.MatchMeta(meta)
			require.NoError(t, err)
			assert.True(t, matches, "Query should match: %s", tt.query)
		})
	}
}

func TestQueryParser_Precedence(t *testing.T) {
	parser := NewQueryParser()

	// Test that AND has higher precedence than OR
	// Query: "a:any=1 OR b:any=2 AND c:any=3"
	// Should be parsed as: "a:any=1 OR (b:any=2 AND c:any=3)"

	// Create metas for testing precedence
	meta1 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "a", Type: property.SearchMetadataTypeString, Value: "1"},
		{Name: "b", Type: property.SearchMetadataTypeString, Value: "0"},
		{Name: "c", Type: property.SearchMetadataTypeString, Value: "0"},
	})

	meta2 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "a", Type: property.SearchMetadataTypeString, Value: "0"},
		{Name: "b", Type: property.SearchMetadataTypeString, Value: "2"},
		{Name: "c", Type: property.SearchMetadataTypeString, Value: "3"},
	})

	meta3 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "a", Type: property.SearchMetadataTypeString, Value: "0"},
		{Name: "b", Type: property.SearchMetadataTypeString, Value: "2"},
		{Name: "c", Type: property.SearchMetadataTypeString, Value: "0"},
	})

	filter, err := parser.Parse("a:any=1 OR b:any=2 AND c:any=3")
	require.NoError(t, err)

	// meta1: a=1, so should match (OR left side is true)
	matches1, err := filter.MatchMeta(meta1)
	require.NoError(t, err)
	assert.True(t, matches1)

	// meta2: a=0 but b=2 AND c=3, so should match (OR right side is true)
	matches2, err := filter.MatchMeta(meta2)
	require.NoError(t, err)
	assert.True(t, matches2)

	// meta3: a=0 and b=2 but c=0, so should not match (both OR sides are false)
	matches3, err := filter.MatchMeta(meta3)
	require.NoError(t, err)
	assert.False(t, matches3)
}

func TestQueryParser_ComplexReal(t *testing.T) {
	parser := NewQueryParser()

	// Test a more realistic complex query with newlines and indentation
	query := `maturity:any=stable,alpha AND 
	          (keywords:all=database,sql OR 
	           (features:any=backup AND features:any=monitoring))`

	filter, err := parser.Parse(query)
	require.NoError(t, err)

	// Should match: stable maturity with database+sql keywords
	meta1 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql", "storage"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false, "monitoring": false}},
	})

	matches1, err := filter.MatchMeta(meta1)
	require.NoError(t, err)
	assert.True(t, matches1)

	// Should match: alpha maturity with backup+monitoring features
	meta2 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "alpha"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true, "monitoring": true}},
	})

	matches2, err := filter.MatchMeta(meta2)
	require.NoError(t, err)
	assert.True(t, matches2)

	// Should not match: beta maturity (first condition fails)
	meta3 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "beta"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true, "monitoring": true}},
	})

	matches3, err := filter.MatchMeta(meta3)
	require.NoError(t, err)
	assert.False(t, matches3)

	// Should not match: stable maturity but neither keyword nor feature conditions met
	meta4 := createMetaWithSearchMetadata([]property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false, "monitoring": true}},
	})

	matches4, err := filter.MatchMeta(meta4)
	require.NoError(t, err)
	assert.False(t, matches4)
}

func TestQueryParser_QuotedFieldNames(t *testing.T) {
	parser := NewQueryParser()

	testCases := []struct {
		name     string
		query    string
		expected map[string][]string
		wantErr  bool
	}{
		{
			name:  "quoted field name with spaces",
			query: `"field name":any=value`,
			expected: map[string][]string{
				"field name": {"value"},
			},
		},
		{
			name:  "quoted field name with colons",
			query: `"field:name":any=value`,
			expected: map[string][]string{
				"field:name": {"value"},
			},
		},
		{
			name:  "quoted field name with equals",
			query: `"field=name":any=value`,
			expected: map[string][]string{
				"field=name": {"value"},
			},
		},
		{
			name:  "quoted field name with commas",
			query: `"field,name":any=value`,
			expected: map[string][]string{
				"field,name": {"value"},
			},
		},
		{
			name:  "quoted field name with escaped quotes",
			query: `"field\"name":any=value`,
			expected: map[string][]string{
				"field\"name": {"value"},
			},
		},
		{
			name:  "quoted field name with backslashes",
			query: `"field\\name":any=value`,
			expected: map[string][]string{
				"field\\name": {"value"},
			},
		},
		{
			name:    "unterminated quoted field name",
			query:   `"field name:any=value`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := parser.Parse(tc.query)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tc.expected), len(filter.criteria))

			for field, expectedValues := range tc.expected {
				found := false
				for _, criterion := range filter.criteria {
					if criterion.name == field {
						assert.ElementsMatch(t, expectedValues, criterion.values)
						found = true
						break
					}
				}
				assert.True(t, found, "field %s not found in criteria", field)
			}
		})
	}
}

func TestQueryParser_QuotedValues(t *testing.T) {
	parser := NewQueryParser()

	testCases := []struct {
		name     string
		query    string
		expected map[string][]string
		wantErr  bool
	}{
		{
			name:  "quoted value with spaces",
			query: `field:any="value with spaces"`,
			expected: map[string][]string{
				"field": {"value with spaces"},
			},
		},
		{
			name:  "quoted value with colons",
			query: `field:any="value:with:colons"`,
			expected: map[string][]string{
				"field": {"value:with:colons"},
			},
		},
		{
			name:  "quoted value with equals",
			query: `field:any="value=with=equals"`,
			expected: map[string][]string{
				"field": {"value=with=equals"},
			},
		},
		{
			name:  "quoted value with commas",
			query: `field:any="value,with,commas"`,
			expected: map[string][]string{
				"field": {"value,with,commas"},
			},
		},
		{
			name:  "multiple quoted values",
			query: `field:any="value one","value two","value three"`,
			expected: map[string][]string{
				"field": {"value one", "value two", "value three"},
			},
		},
		{
			name:  "mixed quoted and unquoted values",
			query: `field:any=unquoted,"quoted value",another`,
			expected: map[string][]string{
				"field": {"unquoted", "quoted value", "another"},
			},
		},
		{
			name:  "quoted value with escaped quotes",
			query: `field:any="value \"with\" quotes"`,
			expected: map[string][]string{
				"field": {"value \"with\" quotes"},
			},
		},
		{
			name:  "quoted value with backslashes",
			query: `field:any="value\\with\\backslashes"`,
			expected: map[string][]string{
				"field": {"value\\with\\backslashes"},
			},
		},
		{
			name:  "quoted value with escape sequences",
			query: `field:any="value\nwith\tescapes"`,
			expected: map[string][]string{
				"field": {"value\nwith\tescapes"},
			},
		},
		{
			name:    "unterminated quoted value",
			query:   `field:any="unterminated value`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := parser.Parse(tc.query)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tc.expected), len(filter.criteria))

			for field, expectedValues := range tc.expected {
				found := false
				for _, criterion := range filter.criteria {
					if criterion.name == field {
						assert.ElementsMatch(t, expectedValues, criterion.values)
						found = true
						break
					}
				}
				assert.True(t, found, "field %s not found in criteria", field)
			}
		})
	}
}

func TestQueryParser_QuotedComplexQueries(t *testing.T) {
	parser := NewQueryParser()

	testCases := []struct {
		name     string
		query    string
		expected map[string][]string
		wantErr  bool
	}{
		{
			name:  "quoted field names and values in complex query",
			query: `"field name":any="value one","value two" AND "other field":all=unquoted,"quoted value"`,
			expected: map[string][]string{
				"field name":  {"value one", "value two"},
				"other field": {"unquoted", "quoted value"},
			},
		},
		{
			name:  "quoted fields with parentheses",
			query: `("field name":any="value one" OR "field name":any="value two") AND category:any=stable`,
			// Note: We can't represent duplicate keys in map, so we'll verify differently
			expected: map[string][]string{
				"category": {"stable"},
			},
		},
		{
			name:  "quoted values with special characters in boolean logic",
			query: `maturity:any="alpha,beta" OR keywords:all="database:sql","key=value"`,
			expected: map[string][]string{
				"maturity": {"alpha,beta"},
				"keywords": {"database:sql", "key=value"},
			},
		},
		{
			name:  "mixed quoting styles",
			query: `"field with spaces":any=unquoted,"quoted value" AND normal_field:all="value with spaces",normal`,
			expected: map[string][]string{
				"field with spaces": {"unquoted", "quoted value"},
				"normal_field":      {"value with spaces", "normal"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := parser.Parse(tc.query)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// For complex queries, we need to check that all expected fields are present
			// Note: Some fields might appear multiple times in complex queries
			expectedFieldCount := 0
			for _, values := range tc.expected {
				expectedFieldCount += len(values)
			}

			// Count actual criteria
			actualCriteriaCount := 0
			for _, criterion := range filter.criteria {
				actualCriteriaCount += len(criterion.values)
			}

			// Verify all expected field/value combinations exist
			for expectedField, expectedValues := range tc.expected {
				for _, expectedValue := range expectedValues {
					found := false
					for _, criterion := range filter.criteria {
						if criterion.name == expectedField {
							for _, actualValue := range criterion.values {
								if actualValue == expectedValue {
									found = true
									break
								}
							}
						}
						if found {
							break
						}
					}
					assert.True(t, found, "expected field %s with value %s not found", expectedField, expectedValue)
				}
			}
		})
	}
}

func TestQueryParser_QuotedFunctionalMatching(t *testing.T) {
	parser := NewQueryParser()

	// Create test operators with special characters in metadata
	type testOperator struct {
		Name     string
		Metadata map[string]interface{}
	}

	operators := []testOperator{
		{
			Name: "operator1",
			Metadata: map[string]interface{}{
				"field name":   []string{"value with spaces", "another value"},
				"field:colon":  []string{"value:with:colons"},
				"field=equals": []string{"value=with=equals"},
				"field,comma":  []string{"value,with,commas"},
				"normal_field": []string{"normal", "values"},
			},
		},
		{
			Name: "operator2",
			Metadata: map[string]interface{}{
				"field name":   []string{"different value"},
				"field:colon":  []string{"other:value"},
				"normal_field": []string{"normal", "other"},
			},
		},
	}

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "quoted field name matching",
			query:    `"field name":any="value with spaces"`,
			expected: []string{"operator1"},
		},
		{
			name:     "quoted field name with colon",
			query:    `"field:colon":any="value:with:colons"`,
			expected: []string{"operator1"},
		},
		{
			name:     "quoted field name with equals",
			query:    `"field=equals":any="value=with=equals"`,
			expected: []string{"operator1"},
		},
		{
			name:     "quoted field name with comma",
			query:    `"field,comma":any="value,with,commas"`,
			expected: []string{"operator1"},
		},
		{
			name:     "quoted value with spaces",
			query:    `"field name":any="value with spaces"`,
			expected: []string{"operator1"},
		},
		{
			name:     "complex query with quoted fields",
			query:    `"field name":any="value with spaces" AND normal_field:any=normal`,
			expected: []string{"operator1"},
		},
		{
			name:     "OR query with quoted fields",
			query:    `"field name":any="value with spaces" OR "field name":any="different value"`,
			expected: []string{"operator1", "operator2"},
		},
	}

	// Helper function to convert testOperator to Meta
	createMetaFromTestOperator := func(op testOperator) declcfg.Meta {
		// Convert metadata to search metadata items
		var searchMetadata []property.SearchMetadataItem
		for key, value := range op.Metadata {
			switch v := value.(type) {
			case string:
				searchMetadata = append(searchMetadata, property.SearchMetadataItem{
					Name:  key,
					Type:  property.SearchMetadataTypeString,
					Value: v,
				})
			case []string:
				searchMetadata = append(searchMetadata, property.SearchMetadataItem{
					Name:  key,
					Type:  property.SearchMetadataTypeListString,
					Value: v,
				})
			case map[string]bool:
				searchMetadata = append(searchMetadata, property.SearchMetadataItem{
					Name:  key,
					Type:  property.SearchMetadataTypeMapStringBoolean,
					Value: v,
				})
			}
		}

		props := []property.Property{
			property.MustBuildPackage("test-package", "1.0.0"),
			property.MustBuildSearchMetadata(searchMetadata),
		}

		type metaBlob struct {
			Properties []property.Property `json:"properties,omitempty"`
		}

		blob := metaBlob{Properties: props}
		blobBytes, _ := json.Marshal(blob)

		return declcfg.Meta{
			Name: op.Name,
			Blob: blobBytes,
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := parser.Parse(tc.query)
			assert.NoError(t, err)

			var matchedNames []string
			for _, op := range operators {
				// Convert testOperator to Meta for testing
				meta := createMetaFromTestOperator(op)
				matches, err := filter.MatchMeta(meta)
				assert.NoError(t, err)

				if matches {
					matchedNames = append(matchedNames, op.Name)
				}
			}

			assert.ElementsMatch(t, tc.expected, matchedNames)
		})
	}
}
