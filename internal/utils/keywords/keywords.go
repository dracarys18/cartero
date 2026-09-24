package keywords

type KeywordWithContext struct {
	Keyword  string   `json:"keyword" toml:"keyword"`
	Context  string   `json:"context_string" toml:"context_string"`
	NotFor   string   `json:"not_for,omitempty" toml:"not_for"`
	Examples []string `json:"examples,omitempty" toml:"examples"`
}
