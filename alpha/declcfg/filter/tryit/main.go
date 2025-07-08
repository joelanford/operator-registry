package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/declcfg/filter"
	"github.com/operator-framework/operator-registry/alpha/property"
)

// sampleOperator represents a sample operator for testing
type sampleOperator struct {
	name        string
	maturity    string
	keywords    []string
	features    map[string]bool
	shouldMatch bool // Expected result for demo queries
}

// createSampleOperators creates a set of sample operators for testing
func createSampleOperators() []sampleOperator {
	return []sampleOperator{
		{
			name:     "database-operator.v1.0.0",
			maturity: "stable",
			keywords: []string{"database", "sql", "storage"},
			features: map[string]bool{"backup": true, "monitoring": true, "replication": false},
		},
		{
			name:     "web-server.v2.0.0",
			maturity: "stable",
			keywords: []string{"web", "http", "server"},
			features: map[string]bool{"ssl": true, "compression": false, "cache": true},
		},
		{
			name:     "cache-operator.v0.8.0",
			maturity: "alpha",
			keywords: []string{"cache", "storage", "memory"},
			features: map[string]bool{"backup": false, "monitoring": true, "distributed": true},
		},
		{
			name:     "backup-service.v1.5.0",
			maturity: "stable",
			keywords: []string{"backup", "storage", "archive"},
			features: map[string]bool{"backup": true, "compression": true, "encryption": true},
		},
		{
			name:     "monitoring-stack.v0.5.0",
			maturity: "beta",
			keywords: []string{"monitoring", "metrics", "observability"},
			features: map[string]bool{"monitoring": true, "alerting": true, "dashboard": false},
		},
		{
			name:     "api-gateway.v1.2.0",
			maturity: "stable",
			keywords: []string{"api", "gateway", "routing"},
			features: map[string]bool{"ssl": true, "rate-limiting": true, "authentication": true},
		},
	}
}

// createMetaFromOperator converts a sampleOperator to a declcfg.Meta
func createMetaFromOperator(op sampleOperator) declcfg.Meta {
	searchMetadata := []property.SearchMetadataItem{
		{
			Name:  "maturity",
			Type:  property.SearchMetadataTypeString,
			Value: op.maturity,
		},
		{
			Name:  "keywords",
			Type:  property.SearchMetadataTypeListString,
			Value: op.keywords,
		},
		{
			Name:  "features",
			Type:  property.SearchMetadataTypeMapStringBoolean,
			Value: op.features,
		},
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
		Name: op.name,
		Blob: blobBytes,
	}
}

// runQuery executes a query against the sample operators and displays results
func runQuery(query string, operators []sampleOperator) {
	fmt.Printf("\n🔍 Query: %s\n", query)
	fmt.Println("----------------------------------------")

	parser := filter.NewQueryParser()
	queryFilter, err := parser.Parse(query)
	if err != nil {
		fmt.Printf("❌ Error parsing query: %v\n", err)
		return
	}

	matchCount := 0
	for _, op := range operators {
		meta := createMetaFromOperator(op)
		matches, err := queryFilter.MatchMeta(meta)
		if err != nil {
			fmt.Printf("❌ Error matching %s: %v\n", op.name, err)
			continue
		}

		if matches {
			matchCount++
			fmt.Printf("✅ %s (maturity: %s, keywords: %v, features: %v)\n",
				op.name, op.maturity, op.keywords, op.features)
		} else {
			fmt.Printf("❌ %s (maturity: %s, keywords: %v, features: %v)\n",
				op.name, op.maturity, op.keywords, op.features)
		}
	}

	fmt.Printf("\n📊 Summary: %d out of %d operators matched\n", matchCount, len(operators))
}

func main() {
	fmt.Println("🚀 Query Parser Demo")
	fmt.Println("===================")

	operators := createSampleOperators()

	// If a query is provided via command line, use it
	if len(os.Args) > 1 {
		query := os.Args[1]
		runQuery(query, operators)
		return
	}

	// Otherwise, run a series of demo queries
	demoQueries := []string{
		// Simple queries
		"maturity:any=stable",
		"keywords:any=database",
		"features:any=backup",

		// Multiple values
		"maturity:any=stable,alpha",
		"keywords:any=database,cache",

		// All operator
		"keywords:all=storage,backup",

		// AND combinations
		"maturity:any=stable AND keywords:any=database",
		"keywords:any=storage AND features:any=backup",

		// OR combinations
		"maturity:any=stable OR features:any=monitoring",
		"keywords:any=database OR keywords:any=cache",

		// Complex combinations
		"maturity:any=stable AND (keywords:any=database OR features:any=backup)",
		"(maturity:any=stable OR maturity:any=alpha) AND features:any=monitoring",

		// Multi-line query (newlines and indentation supported)
		`maturity:any=stable AND
		 (keywords:all=database,sql OR
		  features:any=backup)`,

		// Advanced examples
		"maturity:any=stable AND keywords:all=storage AND features:any=backup",
		"keywords:any=web,api AND features:any=ssl",
	}

	for _, query := range demoQueries {
		runQuery(query, operators)
	}

	fmt.Println("\n💡 Usage Tips:")
	fmt.Println("- Use :any for OR matching within a field (e.g., maturity:any=stable,alpha)")
	fmt.Println("- Use :all for AND matching within a field (e.g., keywords:all=database,sql)")
	fmt.Println("- Use AND/OR to combine multiple field conditions")
	fmt.Println("- Use parentheses for complex logic: (condition1 OR condition2) AND condition3")
	fmt.Println("- Multi-line queries supported: newlines and indentation are normalized to spaces")
	fmt.Println("- Field names: maturity, keywords, features")
	fmt.Println("- Run with a custom query: go run main.go \"your:query=here\"")
}
