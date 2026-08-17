package template

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Item struct {
	ID           string `json:"item_id"`
	Prompt       string `json:"prompt"`
	ResponseType string `json:"response_type"`
	Required     bool   `json:"required"`
	Critical     bool   `json:"critical"`
	AssetType    string `json:"asset_type"`
	SectionID    string `json:"section_id"`
}

type Section struct {
	ID          string `json:"section_id"`
	Title       string `json:"title"`
	Items       []Item `json:"items"`
	SubTemplate string `json:"sub_template,omitempty"`
}

type Definition struct {
	Code        string    `json:"template_code"`
	Version     int       `json:"version"`
	AssetType   string    `json:"asset_type"`
	Title       string    `json:"title"`
	Sections    []Section `json:"sections"`
	Expressions []string  `json:"expressions,omitempty"`
}

func (d Definition) Validate() error {
	if strings.TrimSpace(d.Code) == "" || d.Version <= 0 || strings.TrimSpace(d.AssetType) == "" {
		return errors.New("template code, positive version, and asset type are required")
	}
	for _, section := range d.Sections {
		if strings.TrimSpace(section.ID) == "" {
			return errors.New("section id is required")
		}
		for _, item := range section.Items {
			if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Prompt) == "" {
				return errors.New("item id and prompt are required")
			}
		}
	}
	return nil
}

type AssetNode struct {
	ID        string         `json:"id"`
	AssetType string         `json:"asset_type"`
	Name      string         `json:"name"`
	Fields    map[string]any `json:"fields,omitempty"`
	Children  []AssetNode    `json:"children,omitempty"`
}

type ChecklistItem struct {
	Item
	AssetID   string `json:"asset_id"`
	AssetName string `json:"asset_name"`
}

type Snapshot struct {
	TemplateCode string          `json:"template_code"`
	Version      int             `json:"version"`
	RootAssetID  string          `json:"root_asset_id"`
	ResolvedAt   time.Time       `json:"resolved_at"`
	Items        []ChecklistItem `json:"items"`
}

func (s Snapshot) ItemsCopy() []ChecklistItem {
	copied := make([]ChecklistItem, len(s.Items))
	copy(copied, s.Items)
	return copied
}

// Registry stores immutable-by-convention template definitions by code and
// version. Resolve copies definitions into a snapshot, so later registry
// changes cannot rewrite an inspection's historical checklist.
type Registry struct{ definitions map[string]Definition }

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Definition)} }

func (r *Registry) Register(definition Definition) error {
	if err := definition.Validate(); err != nil {
		return err
	}
	if r.definitions == nil {
		r.definitions = make(map[string]Definition)
	}
	r.definitions[key(definition.Code, definition.Version)] = cloneDefinition(definition)
	return nil
}

func (r *Registry) Resolve(code string, version int, root AssetNode) (Snapshot, error) {
	definition, ok := r.definitions[key(code, version)]
	if !ok {
		return Snapshot{}, fmt.Errorf("template %s version %d not found", code, version)
	}
	if strings.TrimSpace(root.ID) == "" || strings.TrimSpace(root.AssetType) == "" {
		return Snapshot{}, errors.New("root asset id and type are required")
	}
	items := make([]ChecklistItem, 0)
	if err := r.resolveNode(definition, root, &items, map[string]bool{}); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{TemplateCode: code, Version: version, RootAssetID: root.ID, ResolvedAt: time.Now().UTC(), Items: items}, nil
}

func (r *Registry) resolveNode(definition Definition, asset AssetNode, items *[]ChecklistItem, visiting map[string]bool) error {
	if visiting[key(definition.Code, definition.Version)] {
		return fmt.Errorf("cyclic template reference at %s version %d", definition.Code, definition.Version)
	}
	if definition.AssetType != asset.AssetType {
		return fmt.Errorf("template asset type %s does not match asset type %s", definition.AssetType, asset.AssetType)
	}
	visiting[key(definition.Code, definition.Version)] = true
	for _, section := range definition.Sections {
		for _, item := range section.Items {
			item.SectionID = section.ID
			item.AssetType = asset.AssetType
			*items = append(*items, ChecklistItem{Item: item, AssetID: asset.ID, AssetName: asset.Name})
		}
		if section.SubTemplate != "" {
			subCode, subVersion, err := parseReference(section.SubTemplate)
			if err != nil {
				return err
			}
			sub, ok := r.definitions[key(subCode, subVersion)]
			if !ok {
				return fmt.Errorf("sub-template %s version %d not found", subCode, subVersion)
			}
			for _, child := range asset.Children {
				if child.AssetType == sub.AssetType {
					if err := r.resolveNode(sub, child, items, visiting); err != nil {
						return err
					}
				}
			}
		}
	}
	delete(visiting, key(definition.Code, definition.Version))
	return nil
}

func parseReference(reference string) (string, int, error) {
	parts := strings.Split(reference, "@")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid template reference %q", reference)
	}
	version, err := strconv.Atoi(parts[1])
	if err != nil || version <= 0 {
		return "", 0, fmt.Errorf("invalid template reference version %q", reference)
	}
	if strings.TrimSpace(parts[0]) == "" {
		return "", 0, errors.New("template reference code is required")
	}
	return parts[0], version, nil
}

func key(code string, version int) string { return fmt.Sprintf("%s@%d", code, version) }

func cloneDefinition(definition Definition) Definition {
	cloned := definition
	cloned.Sections = make([]Section, len(definition.Sections))
	for index, section := range definition.Sections {
		cloned.Sections[index] = section
		cloned.Sections[index].Items = append([]Item(nil), section.Items...)
	}
	cloned.Expressions = append([]string(nil), definition.Expressions...)
	return cloned
}

// Evaluate evaluates boolean expressions over a flat field map. It supports
// &&, ||, ==, !=, >, >=, <, <=, parentheses, strings, booleans, numbers, and
// field identifiers. It intentionally has no side effects or external calls.
func Evaluate(expression string, fields map[string]any) (bool, error) {
	parser := newParser(expression, fields)
	result, err := parser.parseOr()
	if err != nil {
		return false, err
	}
	if parser.current.kind != tokenEOF {
		return false, fmt.Errorf("unexpected token %q", parser.current.text)
	}
	return result, nil
}

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenWord
	tokenString
	tokenNumber
	tokenOperator
	tokenLParen
	tokenRParen
)

type token struct {
	kind tokenKind
	text string
}

type lexer struct {
	input    []rune
	position int
}

func (l *lexer) next() token {
	for l.position < len(l.input) && unicode.IsSpace(l.input[l.position]) {
		l.position++
	}
	if l.position >= len(l.input) {
		return token{kind: tokenEOF}
	}
	start := l.position
	char := l.input[l.position]
	if char == '(' {
		l.position++
		return token{kind: tokenLParen, text: "("}
	}
	if char == ')' {
		l.position++
		return token{kind: tokenRParen, text: ")"}
	}
	if char == '"' || char == '\'' {
		quote := char
		l.position++
		start = l.position
		for l.position < len(l.input) && l.input[l.position] != quote {
			l.position++
		}
		text := string(l.input[start:l.position])
		if l.position < len(l.input) {
			l.position++
		}
		return token{kind: tokenString, text: text}
	}
	if strings.ContainsRune("<>!=", char) {
		l.position++
		if l.position < len(l.input) && l.input[l.position] == '=' {
			l.position++
		}
		return token{kind: tokenOperator, text: string(l.input[start:l.position])}
	}
	if char == '&' || char == '|' {
		l.position++
		if l.position < len(l.input) && l.input[l.position] == char {
			l.position++
		}
		return token{kind: tokenOperator, text: string(l.input[start:l.position])}
	}
	for l.position < len(l.input) && !unicode.IsSpace(l.input[l.position]) && !strings.ContainsRune("()<>!=&|", l.input[l.position]) {
		l.position++
	}
	text := string(l.input[start:l.position])
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return token{kind: tokenNumber, text: text}
	}
	return token{kind: tokenWord, text: text}
}

type parser struct {
	lexer   lexer
	current token
	fields  map[string]any
}

func newParser(input string, fields map[string]any) *parser {
	p := &parser{lexer: lexer{input: []rune(input)}, fields: fields}
	p.current = p.lexer.next()
	return p
}
func (p *parser) advance() { p.current = p.lexer.next() }
func (p *parser) parseOr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return false, err
	}
	for p.current.text == "||" {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return false, err
		}
		left = left || right
	}
	return left, nil
}
func (p *parser) parseAnd() (bool, error) {
	left, err := p.parseComparison()
	if err != nil {
		return false, err
	}
	for p.current.text == "&&" {
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return false, err
		}
		left = left && right
	}
	return left, nil
}
func (p *parser) parseComparison() (bool, error) {
	if p.current.kind == tokenLParen {
		p.advance()
		value, err := p.parseOr()
		if err != nil {
			return false, err
		}
		if p.current.kind != tokenRParen {
			return false, errors.New("missing closing parenthesis")
		}
		p.advance()
		return value, nil
	}
	left, err := p.value()
	if err != nil {
		return false, err
	}
	if p.current.kind != tokenOperator {
		if boolean, ok := left.(bool); ok {
			return boolean, nil
		}
		return false, errors.New("expression must resolve to boolean")
	}
	operator := p.current.text
	p.advance()
	right, err := p.value()
	if err != nil {
		return false, err
	}
	return compare(left, operator, right)
}
func (p *parser) value() (any, error) {
	current := p.current
	p.advance()
	switch current.kind {
	case tokenString:
		return current.text, nil
	case tokenNumber:
		return strconv.ParseFloat(current.text, 64)
	case tokenWord:
		switch current.text {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		value, ok := p.fields[current.text]
		if !ok {
			return nil, fmt.Errorf("unknown field %q", current.text)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("expected value, got %q", current.text)
	}
}
func compare(left any, operator string, right any) (bool, error) {
	if leftNumber, ok := numeric(left); ok {
		if rightNumber, ok := numeric(right); ok {
			return compareNumbers(leftNumber, operator, rightNumber)
		}
	}
	leftString, leftOK := left.(string)
	rightString, rightOK := right.(string)
	if leftOK && rightOK {
		switch operator {
		case "==":
			return leftString == rightString, nil
		case "!=":
			return leftString != rightString, nil
		}
	}
	leftBool, leftOK := left.(bool)
	rightBool, rightOK := right.(bool)
	if leftOK && rightOK {
		switch operator {
		case "==":
			return leftBool == rightBool, nil
		case "!=":
			return leftBool != rightBool, nil
		}
	}
	return false, fmt.Errorf("operator %s is not valid for supplied values", operator)
}
func numeric(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	}
	return 0, false
}
func compareNumbers(left float64, operator string, right float64) (bool, error) {
	switch operator {
	case "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	case ">":
		return left > right, nil
	case ">=":
		return left >= right, nil
	case "<":
		return left < right, nil
	case "<=":
		return left <= right, nil
	}
	return false, fmt.Errorf("unknown numeric operator %s", operator)
}
