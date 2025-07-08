package filter_test

import (
	"fmt"
	"log"

	"github.com/operator-framework/operator-registry/alpha/declcfg/filter"
	"github.com/operator-framework/operator-registry/alpha/property"
)

// ExampleQueryParser_Parse demonstrates basic usage of the query parser
func ExampleQueryParser_Parse() {
	// Create a query parser
	parser := filter.NewQueryParser()

	// Simple query: find operators that are stable OR alpha
	query := "maturity:any=stable,alpha"

	queryFilter, err := parser.Parse(query)
	if err != nil {
		log.Fatal(err)
	}

	// Create sample metadata to test against
	stableMeta := createMetaWithSearchMetadata("stable-operator.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
	})

	alphaMeta := createMetaWithSearchMetadata("alpha-operator.v0.5.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "alpha"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http"}},
	})

	betaMeta := createMetaWithSearchMetadata("beta-operator.v0.8.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "beta"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"cache", "storage"}},
	})

	// Test the filter
	stableMatch, _ := queryFilter.MatchMeta(stableMeta)
	alphaMatch, _ := queryFilter.MatchMeta(alphaMeta)
	betaMatch, _ := queryFilter.MatchMeta(betaMeta)

	fmt.Printf("Stable operator matches: %t\n", stableMatch)
	fmt.Printf("Alpha operator matches: %t\n", alphaMatch)
	fmt.Printf("Beta operator matches: %t\n", betaMatch)

	// Output:
	// Stable operator matches: true
	// Alpha operator matches: true
	// Beta operator matches: false
}

// ExampleQueryParser_Parse_complex demonstrates complex queries with AND/OR logic
func ExampleQueryParser_Parse_complex() {
	parser := filter.NewQueryParser()

	// Complex query: find stable operators with database capabilities OR any operator with backup features
	query := "maturity:any=stable AND keywords:any=database OR features:any=backup"

	queryFilter, err := parser.Parse(query)
	if err != nil {
		log.Fatal(err)
	}

	// Test metas
	stableDatabaseMeta := createMetaWithSearchMetadata("stable-db.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false, "monitoring": true}},
	})

	alphaBackupMeta := createMetaWithSearchMetadata("alpha-backup.v0.5.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "alpha"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"storage", "backup"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true, "monitoring": false}},
	})

	betaWebMeta := createMetaWithSearchMetadata("beta-web.v0.8.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "beta"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false, "ssl": true}},
	})

	// Test the filter
	stableDbMatch, _ := queryFilter.MatchMeta(stableDatabaseMeta)
	alphaBackupMatch, _ := queryFilter.MatchMeta(alphaBackupMeta)
	betaWebMatch, _ := queryFilter.MatchMeta(betaWebMeta)

	fmt.Printf("Stable database operator matches: %t\n", stableDbMatch)
	fmt.Printf("Alpha backup operator matches: %t\n", alphaBackupMatch)
	fmt.Printf("Beta web operator matches: %t\n", betaWebMatch)

	// Output:
	// Stable database operator matches: true
	// Alpha backup operator matches: true
	// Beta web operator matches: false
}

// ExampleQueryParser_Parse_parentheses demonstrates using parentheses for complex logic
func ExampleQueryParser_Parse_parentheses() {
	parser := filter.NewQueryParser()

	// Query with parentheses: (stable OR alpha) AND (database capabilities OR backup features)
	query := "(maturity:any=stable,alpha) AND (keywords:any=database OR features:any=backup)"

	queryFilter, err := parser.Parse(query)
	if err != nil {
		log.Fatal(err)
	}

	// Test different combinations
	stableDatabase := createMetaWithSearchMetadata("stable-db.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false}},
	})

	alphaBackup := createMetaWithSearchMetadata("alpha-backup.v0.5.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "alpha"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"storage"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true}},
	})

	betaDatabase := createMetaWithSearchMetadata("beta-db.v0.8.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "beta"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false}},
	})

	stableWeb := createMetaWithSearchMetadata("stable-web.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false}},
	})

	// Test matches
	stableDbMatch, _ := queryFilter.MatchMeta(stableDatabase)
	alphaBackupMatch, _ := queryFilter.MatchMeta(alphaBackup)
	betaDbMatch, _ := queryFilter.MatchMeta(betaDatabase)
	stableWebMatch, _ := queryFilter.MatchMeta(stableWeb)

	fmt.Printf("Stable database: %t (stable maturity ✓, database keyword ✓)\n", stableDbMatch)
	fmt.Printf("Alpha backup: %t (alpha maturity ✓, backup feature ✓)\n", alphaBackupMatch)
	fmt.Printf("Beta database: %t (beta maturity ✗, database keyword ✓)\n", betaDbMatch)
	fmt.Printf("Stable web: %t (stable maturity ✓, no database/backup ✗)\n", stableWebMatch)

	// Output:
	// Stable database: true (stable maturity ✓, database keyword ✓)
	// Alpha backup: true (alpha maturity ✓, backup feature ✓)
	// Beta database: false (beta maturity ✗, database keyword ✓)
	// Stable web: false (stable maturity ✓, no database/backup ✗)
}

// ExampleQueryParser_Parse_allOperator demonstrates the "all" operator for list matching
func ExampleQueryParser_Parse_allOperator() {
	parser := filter.NewQueryParser()

	// Query: find operators that have ALL of these keywords: database, sql, storage
	query := "keywords:all=database,sql,storage"

	queryFilter, err := parser.Parse(query)
	if err != nil {
		log.Fatal(err)
	}

	// Test metas with different keyword combinations
	fullFeatureMeta := createMetaWithSearchMetadata("full-db.v1.0.0", []property.SearchMetadataItem{
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql", "storage", "backup"}},
	})

	partialMeta := createMetaWithSearchMetadata("partial-db.v1.0.0", []property.SearchMetadataItem{
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
	})

	differentMeta := createMetaWithSearchMetadata("web-server.v1.0.0", []property.SearchMetadataItem{
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http", "server"}},
	})

	// Test matches
	fullMatch, _ := queryFilter.MatchMeta(fullFeatureMeta)
	partialMatch, _ := queryFilter.MatchMeta(partialMeta)
	differentMatch, _ := queryFilter.MatchMeta(differentMeta)

	fmt.Printf("Full feature operator: %t (has all: database, sql, storage)\n", fullMatch)
	fmt.Printf("Partial operator: %t (missing: storage)\n", partialMatch)
	fmt.Printf("Different operator: %t (has: web, http, server)\n", differentMatch)

	// Output:
	// Full feature operator: true (has all: database, sql, storage)
	// Partial operator: false (missing: storage)
	// Different operator: false (has: web, http, server)
}

// ExampleQueryParser_MustParse demonstrates the MustParse convenience method
func ExampleQueryParser_MustParse() {
	parser := filter.NewQueryParser()

	// For known-good queries, you can use MustParse which panics on error
	queryFilter := parser.MustParse("maturity:any=stable,alpha")

	// Use the filter
	meta := createMetaWithSearchMetadata("test-operator.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
	})

	matches, _ := queryFilter.MatchMeta(meta)
	fmt.Printf("Operator matches: %t\n", matches)

	// Output:
	// Operator matches: true
}

// ExampleQueryParser_Parse_multipleFields demonstrates filtering on multiple different fields
func ExampleQueryParser_Parse_multipleFields() {
	parser := filter.NewQueryParser()

	// Query: stable maturity AND database keywords AND backup features
	query := "maturity:any=stable AND keywords:any=database AND features:any=backup"

	queryFilter, err := parser.Parse(query)
	if err != nil {
		log.Fatal(err)
	}

	// Test different operators
	perfectMatch := createMetaWithSearchMetadata("perfect-operator.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true, "monitoring": false}},
	})

	noBackup := createMetaWithSearchMetadata("no-backup.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"database", "sql"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": false, "monitoring": true}},
	})

	noDatabase := createMetaWithSearchMetadata("no-database.v1.0.0", []property.SearchMetadataItem{
		{Name: "maturity", Type: property.SearchMetadataTypeString, Value: "stable"},
		{Name: "keywords", Type: property.SearchMetadataTypeListString, Value: []string{"web", "http"}},
		{Name: "features", Type: property.SearchMetadataTypeMapStringBoolean, Value: map[string]bool{"backup": true, "ssl": true}},
	})

	// Test matches
	perfectMatch_, _ := queryFilter.MatchMeta(perfectMatch)
	noBackupMatch, _ := queryFilter.MatchMeta(noBackup)
	noDatabaseMatch, _ := queryFilter.MatchMeta(noDatabase)

	fmt.Printf("Perfect match: %t (stable ✓, database ✓, backup ✓)\n", perfectMatch_)
	fmt.Printf("No backup: %t (stable ✓, database ✓, backup ✗)\n", noBackupMatch)
	fmt.Printf("No database: %t (stable ✓, database ✗, backup ✓)\n", noDatabaseMatch)

	// Output:
	// Perfect match: true (stable ✓, database ✓, backup ✓)
	// No backup: false (stable ✓, database ✓, backup ✗)
	// No database: false (stable ✓, database ✗, backup ✓)
}

// ExampleQueryParser_Parse_quotedFields demonstrates using quoted field names and values
// to handle special characters like spaces, colons, equals signs, and commas.
func ExampleQueryParser_Parse_quotedFields() {
	parser := filter.NewQueryParser()

	// Example 1: Quoted field names with special characters
	_, err := parser.Parse(`"field name":any=value`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Quoted field name with spaces parsed successfully")

	// Example 2: Quoted values with special characters
	_, err = parser.Parse(`field:any="value with spaces","value:with:colons","value=with=equals"`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Quoted values with special characters parsed successfully")

	// Example 3: Mixed quoted and unquoted
	_, err = parser.Parse(`"field name":any=unquoted,"quoted value" AND normal_field:all="value with spaces",normal`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Mixed quoted and unquoted fields/values parsed successfully")

	// Example 4: Escaped quotes within quoted strings
	_, err = parser.Parse(`field:any="value \"with\" quotes"`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Escaped quotes within quoted strings parsed successfully")

	// Output:
	// ✓ Quoted field name with spaces parsed successfully
	// ✓ Quoted values with special characters parsed successfully
	// ✓ Mixed quoted and unquoted fields/values parsed successfully
	// ✓ Escaped quotes within quoted strings parsed successfully
}

// ExampleQueryParser_Parse_quotedVsUnquoted demonstrates the difference between
// quoted and unquoted values when dealing with special characters.
func ExampleQueryParser_Parse_quotedVsUnquoted() {
	parser := filter.NewQueryParser()

	// Unquoted comma-separated values - treated as multiple values
	_, err := parser.Parse(`keywords:any=database,sql,storage`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Unquoted comma-separated values: database, sql, storage (3 separate values)")

	// Quoted comma-separated values - treated as single value
	_, err = parser.Parse(`keywords:any="database,sql,storage"`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Quoted comma-separated values: \"database,sql,storage\" (1 single value)")

	// Mixed quoted and unquoted values
	_, err = parser.Parse(`keywords:any=unquoted,"quoted,value",another`)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Mixed values: unquoted, \"quoted,value\", another (3 values total)")

	// Output:
	// ✓ Unquoted comma-separated values: database, sql, storage (3 separate values)
	// ✓ Quoted comma-separated values: "database,sql,storage" (1 single value)
	// ✓ Mixed values: unquoted, "quoted,value", another (3 values total)
}
