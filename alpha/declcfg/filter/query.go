// Package filter provides functionality for filtering File-Based Catalog metadata
// based on search metadata properties with a query language parser.
package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// QueryParser parses query strings into Filter objects.
//
// Query Language Syntax:
//
//	field:operator=value1,value2,... [AND/OR field:operator=value1,value2,...]
//
// Where:
//   - field: metadata field name
//   - operator: "any" (hasAny) or "all" (hasAll)
//   - values: comma-separated list of values to match
//   - AND/OR: logical operators (default is AND if not specified)
//
// Examples:
//   - "maturity:any=stable,alpha"
//   - "keywords:all=database,sql"
//   - "maturity:any=stable AND keywords:all=database,sql"
//   - "maturity:any=stable OR features:any=backup"
//   - "maturity:any=stable AND (keywords:all=database OR features:any=backup)"
type QueryParser struct{}

// NewQueryParser creates a new QueryParser instance.
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// Parse parses a query string and returns a Filter.
//
// The query string follows the syntax:
//
//	field:operator=value1,value2,... [AND/OR field:operator=value1,value2,...]
//
// Returns an error if the query string is malformed.
func (p *QueryParser) Parse(query string) (*Filter, error) {
	if strings.TrimSpace(query) == "" {
		return New(All), nil
	}

	// Parse the query into an AST
	ast, err := p.parseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	// Convert AST to Filter
	return p.astToFilter(ast)
}

// queryAST represents the abstract syntax tree for a query
type queryAST interface {
	isQueryAST()
}

// criterionAST represents a single criterion (field:operator=values)
type criterionAST struct {
	field    string
	operator string
	values   []string
}

func (c *criterionAST) isQueryAST() {}

// binaryOpAST represents a binary operation (AND/OR)
type binaryOpAST struct {
	left     queryAST
	operator string // "AND" or "OR"
	right    queryAST
}

func (b *binaryOpAST) isQueryAST() {}

// parseQuery parses a query string into an AST
func (p *QueryParser) parseQuery(query string) (queryAST, error) {
	// Normalize whitespace: replace all whitespace sequences (including newlines) with single spaces
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")
	query = strings.TrimSpace(query)

	// Simple recursive descent parser
	return p.parseOrExpression(query)
}

// parseOrExpression parses OR expressions (lowest precedence)
func (p *QueryParser) parseOrExpression(query string) (queryAST, error) {
	left, remaining, err := p.parseAndExpression(query)
	if err != nil {
		return nil, err
	}

	remaining = strings.TrimSpace(remaining)
	if remaining == "" {
		return left, nil
	}

	// Look for OR operator
	orRegex := regexp.MustCompile(`^(?i)\s*OR\s+(.*)$`)
	if matches := orRegex.FindStringSubmatch(remaining); matches != nil {
		right, err := p.parseOrExpression(matches[1])
		if err != nil {
			return nil, err
		}
		return &binaryOpAST{
			left:     left,
			operator: "OR",
			right:    right,
		}, nil
	}

	return left, nil
}

// parseAndExpression parses AND expressions (higher precedence than OR)
func (p *QueryParser) parseAndExpression(query string) (queryAST, string, error) {
	left, remaining, err := p.parsePrimary(query)
	if err != nil {
		return nil, "", err
	}

	remaining = strings.TrimSpace(remaining)
	if remaining == "" {
		return left, "", nil
	}

	// Look for AND operator (explicit or implicit)
	andRegex := regexp.MustCompile(`^(?i)\s*AND\s+(.*)$`)

	// For implicit AND, we need to check if the remaining string starts with a valid criterion
	// This is more complex now because we support quoted field names
	var nextExpr string
	if matches := andRegex.FindStringSubmatch(remaining); matches != nil {
		nextExpr = matches[1]
	} else if p.startsWithCriterion(remaining) {
		// Implicit AND (no operator specified)
		nextExpr = remaining
	} else {
		return left, remaining, nil
	}

	right, finalRemaining, err := p.parseAndExpression(nextExpr)
	if err != nil {
		return nil, "", err
	}

	return &binaryOpAST{
		left:     left,
		operator: "AND",
		right:    right,
	}, finalRemaining, nil
}

// startsWithCriterion checks if a string starts with a valid criterion
func (p *QueryParser) startsWithCriterion(s string) bool {
	s = strings.TrimSpace(s)

	// Check for quoted field name
	if strings.HasPrefix(s, `"`) {
		return true
	}

	// Check for unquoted field name (alphanumeric + underscore)
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*:`).MatchString(s)
}

// parsePrimary parses primary expressions (criteria or parenthesized expressions)
func (p *QueryParser) parsePrimary(query string) (queryAST, string, error) {
	query = strings.TrimSpace(query)

	// Handle parentheses
	if strings.HasPrefix(query, "(") {
		// Find matching closing parenthesis
		depth := 1
		i := 1
		for i < len(query) && depth > 0 {
			if query[i] == '(' {
				depth++
			} else if query[i] == ')' {
				depth--
			}
			i++
		}
		if depth > 0 {
			return nil, "", fmt.Errorf("mismatched parentheses in query")
		}

		// Parse the expression inside parentheses
		inner := query[1 : i-1]
		innerAST, err := p.parseQuery(inner)
		if err != nil {
			return nil, "", err
		}

		return innerAST, query[i:], nil
	}

	// Parse criterion: field:operator=value1,value2,...
	// Now supports quoted field names and values
	return p.parseCriterion(query)
}

// parseCriterion parses a single criterion with support for quoted field names and values
func (p *QueryParser) parseCriterion(query string) (queryAST, string, error) {
	query = strings.TrimSpace(query)

	// Parse field name (quoted or unquoted)
	field, remaining, err := p.parseFieldName(query)
	if err != nil {
		return nil, "", err
	}

	// Expect colon
	remaining = strings.TrimSpace(remaining)
	if !strings.HasPrefix(remaining, ":") {
		return nil, "", fmt.Errorf("expected ':' after field name")
	}
	remaining = remaining[1:]

	// Parse operator
	remaining = strings.TrimSpace(remaining)
	operatorRegex := regexp.MustCompile(`^(?i)(any|all)=(.*)$`)
	matches := operatorRegex.FindStringSubmatch(remaining)
	if matches == nil {
		return nil, "", fmt.Errorf("expected operator (any|all) followed by '='")
	}

	operator := strings.ToLower(matches[1])
	valuesStr := matches[2]

	// Parse values (comma-separated, with support for quoted values)
	values, finalRemaining, err := p.parseValueList(valuesStr)
	if err != nil {
		return nil, "", err
	}

	if len(values) == 0 {
		return nil, "", fmt.Errorf("no values specified for criterion: %s", field)
	}

	criterion := &criterionAST{
		field:    field,
		operator: operator,
		values:   values,
	}

	return criterion, finalRemaining, nil
}

// parseFieldName parses a field name (quoted or unquoted)
func (p *QueryParser) parseFieldName(query string) (string, string, error) {
	if strings.HasPrefix(query, `"`) {
		// Quoted field name
		return p.parseQuotedString(query)
	}

	// Unquoted field name
	fieldRegex := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)(.*)$`)
	matches := fieldRegex.FindStringSubmatch(query)
	if matches == nil {
		return "", "", fmt.Errorf("invalid field name format")
	}

	return matches[1], matches[2], nil
}

// parseValueList parses a comma-separated list of values with support for quoted values
func (p *QueryParser) parseValueList(valuesStr string) ([]string, string, error) {
	values := make([]string, 0)
	remaining := valuesStr

	for {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		var value string
		var err error

		if strings.HasPrefix(remaining, `"`) {
			// Quoted value
			value, remaining, err = p.parseQuotedString(remaining)
			if err != nil {
				return nil, "", err
			}
		} else {
			// Unquoted value - parse until comma, space, or end of criterion
			valueRegex := regexp.MustCompile(`^([^,\s)]+)(.*)$`)
			matches := valueRegex.FindStringSubmatch(remaining)
			if matches == nil {
				return nil, "", fmt.Errorf("invalid value format")
			}
			value = matches[1]
			remaining = matches[2]
		}

		values = append(values, value)

		// Check for comma separator
		remaining = strings.TrimSpace(remaining)
		if strings.HasPrefix(remaining, ",") {
			remaining = remaining[1:]
			continue
		}

		// No comma, we're done with this value list
		break
	}

	return values, remaining, nil
}

// parseQuotedString parses a quoted string with escape sequence support
func (p *QueryParser) parseQuotedString(query string) (string, string, error) {
	if !strings.HasPrefix(query, `"`) {
		return "", "", fmt.Errorf("expected quoted string")
	}

	var result strings.Builder
	i := 1 // Skip opening quote

	for i < len(query) {
		ch := query[i]

		if ch == '"' {
			// End of quoted string
			return result.String(), query[i+1:], nil
		}

		if ch == '\\' && i+1 < len(query) {
			// Escape sequence
			next := query[i+1]
			switch next {
			case '"':
				result.WriteByte('"')
			case '\\':
				result.WriteByte('\\')
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			default:
				// Unknown escape, keep both characters
				result.WriteByte('\\')
				result.WriteByte(next)
			}
			i += 2
		} else {
			result.WriteByte(ch)
			i++
		}
	}

	return "", "", fmt.Errorf("unterminated quoted string")
}

// astToFilter converts an AST to a Filter
func (p *QueryParser) astToFilter(ast queryAST) (*Filter, error) {
	switch node := ast.(type) {
	case *criterionAST:
		filter := New(All)
		switch node.operator {
		case "any":
			filter.HasAny(node.field, node.values...)
		case "all":
			filter.HasAll(node.field, node.values...)
		default:
			return nil, fmt.Errorf("unknown operator: %s", node.operator)
		}
		return filter, nil

	case *binaryOpAST:
		// For binary operations, we need to create a custom filter that combines
		// the logic from both sub-filters
		leftFilter, err := p.astToFilter(node.left)
		if err != nil {
			return nil, err
		}
		rightFilter, err := p.astToFilter(node.right)
		if err != nil {
			return nil, err
		}

		// Create a combined filter that contains all criteria from both sides
		combinedFilter := &Filter{
			criteria: append(leftFilter.criteria, rightFilter.criteria...),
		}

		// Set the match function based on the operator
		switch node.operator {
		case "AND":
			combinedFilter.matchFunc = func(results []Result) bool {
				// Split results by position to match left and right filters
				leftResults := results[:len(leftFilter.criteria)]
				rightResults := results[len(leftFilter.criteria):]

				// Both sub-filters must match
				return leftFilter.matchFunc(leftResults) && rightFilter.matchFunc(rightResults)
			}
		case "OR":
			combinedFilter.matchFunc = func(results []Result) bool {
				// Split results by position to match left and right filters
				leftResults := results[:len(leftFilter.criteria)]
				rightResults := results[len(leftFilter.criteria):]

				// Either sub-filter can match
				return leftFilter.matchFunc(leftResults) || rightFilter.matchFunc(rightResults)
			}
		default:
			return nil, fmt.Errorf("unknown binary operator: %s", node.operator)
		}

		return combinedFilter, nil

	default:
		return nil, fmt.Errorf("unknown AST node type: %T", node)
	}
}

// MustParse parses a query string and panics if there's an error.
// This is useful for testing and when you're sure the query is valid.
func (p *QueryParser) MustParse(query string) *Filter {
	filter, err := p.Parse(query)
	if err != nil {
		panic(fmt.Sprintf("failed to parse query: %v", err))
	}
	return filter
}
