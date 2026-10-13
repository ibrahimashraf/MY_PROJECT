package completeness

import "strings"

type Requirement struct {
	Field     string `json:"field"`
	Mandatory bool   `json:"mandatory"`
}

type Suggestion struct {
	Field    string `json:"field"`
	Reason   string `json:"reason"`
	Blocking bool   `json:"blocking"`
}

type Result struct {
	Complete    bool         `json:"complete"`
	Missing     []string     `json:"missing"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
}

// Check is deterministic and authoritative for mandatory data presence. It
// never invokes AI and never mutates the supplied input map.
func Check(requirements []Requirement, data map[string]any) Result {
	missing := make([]string, 0)
	for _, requirement := range requirements {
		if requirement.Mandatory && !present(data, requirement.Field) {
			missing = append(missing, requirement.Field)
		}
	}
	return Result{Complete: len(missing) == 0, Missing: missing}
}

// AddAdvisorySuggestions adds non-blocking guidance after deterministic
// completeness has run. Suggestions cannot change Complete or Missing.
func AddAdvisorySuggestions(result Result, suggestions []Suggestion) Result {
	copied := Result{Complete: result.Complete, Missing: append([]string(nil), result.Missing...), Suggestions: make([]Suggestion, len(suggestions))}
	for index, suggestion := range suggestions {
		suggestion.Blocking = false
		copied.Suggestions[index] = suggestion
	}
	return copied
}

func present(data map[string]any, path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	current := any(data)
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return false
		}
		value, ok := object[part]
		if !ok || value == nil {
			return false
		}
		current = value
	}
	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value) != ""
	case []string:
		return len(value) > 0
	case []any:
		return len(value) > 0
	default:
		return true
	}
}
