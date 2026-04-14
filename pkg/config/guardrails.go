package config

import "fmt"

// GuardrailsConfig contains guardrails configuration.
// Guardrails provide safety controls for agent inputs, outputs, and tool calls.
type GuardrailsConfig struct {
	// Enabled controls whether guardrails are active.
	Enabled bool `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,description=Whether guardrails are active,default=true"`

	// Input guardrail configurations.
	Input *InputGuardrailsConfig `yaml:"input,omitempty" json:"input,omitempty" jsonschema:"title=Input Guardrails,description=Input validation and sanitization settings"`

	// Output guardrail configurations.
	Output *OutputGuardrailsConfig `yaml:"output,omitempty" json:"output,omitempty" jsonschema:"title=Output Guardrails,description=Output filtering and redaction settings"`

	// Tool guardrail configurations.
	Tool *ToolGuardrailsConfig `yaml:"tool,omitempty" json:"tool,omitempty" jsonschema:"title=Tool Guardrails,description=Tool authorization settings"`

	// Moderation configures LLM-powered content moderation.
	// Adds semantic understanding beyond regex-based guardrails.
	Moderation *ModerationGuardrailConfig `yaml:"moderation,omitempty" json:"moderation,omitempty" jsonschema:"title=LLM Moderation,description=LLM-powered content moderation settings"`
}

// InputGuardrailsConfig contains input guardrail settings.
type InputGuardrailsConfig struct {
	// ChainMode for input guardrails: "fail_fast" or "collect_all".
	ChainMode string `yaml:"chain_mode,omitempty" json:"chain_mode,omitempty" jsonschema:"title=Chain Mode,description=How to handle multiple guardrails,enum=fail_fast,enum=collect_all,default=fail_fast"`

	// Length validation settings.
	Length *LengthGuardrailConfig `yaml:"length,omitempty" json:"length,omitempty" jsonschema:"title=Length Validation,description=Input length constraints"`

	// Injection detection settings.
	Injection *InjectionGuardrailConfig `yaml:"injection,omitempty" json:"injection,omitempty" jsonschema:"title=Injection Detection,description=Prompt injection protection"`

	// Sanitization settings.
	Sanitizer *SanitizerGuardrailConfig `yaml:"sanitizer,omitempty" json:"sanitizer,omitempty" jsonschema:"title=Input Sanitization,description=Input cleaning and normalization"`

	// Pattern validation settings.
	Pattern *PatternGuardrailConfig `yaml:"pattern,omitempty" json:"pattern,omitempty" jsonschema:"title=Pattern Validation,description=Regex-based validation"`
}

// LengthGuardrailConfig configures input length validation.
type LengthGuardrailConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=true"`
	MinLength int    `yaml:"min_length,omitempty" json:"min_length,omitempty" jsonschema:"title=Minimum Length,default=1"`
	MaxLength int    `yaml:"max_length,omitempty" json:"max_length,omitempty" jsonschema:"title=Maximum Length,default=100000"`
	Action    string `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=allow,enum=block,enum=warn,default=block"`
	Severity  string `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"title=Severity,enum=low,enum=medium,enum=high,enum=critical,default=medium"`
}

// InjectionGuardrailConfig configures prompt injection detection.
type InjectionGuardrailConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=true"`
	Patterns      []string `yaml:"patterns,omitempty" json:"patterns,omitempty" jsonschema:"title=Custom Patterns,description=Additional regex patterns to detect"`
	CaseSensitive bool     `yaml:"case_sensitive,omitempty" json:"case_sensitive,omitempty" jsonschema:"title=Case Sensitive,default=false"`
	Action        string   `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=allow,enum=block,enum=warn,default=block"`
	Severity      string   `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"title=Severity,enum=low,enum=medium,enum=high,enum=critical,default=high"`
}

// SanitizerGuardrailConfig configures input sanitization.
type SanitizerGuardrailConfig struct {
	Enabled          bool `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=true"`
	TrimWhitespace   bool `yaml:"trim_whitespace,omitempty" json:"trim_whitespace,omitempty" jsonschema:"title=Trim Whitespace,default=true"`
	NormalizeUnicode bool `yaml:"normalize_unicode,omitempty" json:"normalize_unicode,omitempty" jsonschema:"title=Normalize Unicode,default=false"`
	MaxLength        int  `yaml:"max_length,omitempty" json:"max_length,omitempty" jsonschema:"title=Max Length,description=Truncate if exceeded (0=no limit)"`
	StripHTML        bool `yaml:"strip_html,omitempty" json:"strip_html,omitempty" jsonschema:"title=Strip HTML,default=false"`
}

// PatternGuardrailConfig configures pattern-based validation.
type PatternGuardrailConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=false"`
	AllowPatterns []string `yaml:"allow_patterns,omitempty" json:"allow_patterns,omitempty" jsonschema:"title=Allow Patterns,description=Patterns that input must match"`
	BlockPatterns []string `yaml:"block_patterns,omitempty" json:"block_patterns,omitempty" jsonschema:"title=Block Patterns,description=Patterns that input must NOT match"`
	Action        string   `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=allow,enum=block,enum=warn,default=block"`
	Severity      string   `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"title=Severity,enum=low,enum=medium,enum=high,enum=critical,default=medium"`
}

// OutputGuardrailsConfig contains output guardrail settings.
type OutputGuardrailsConfig struct {
	// ChainMode for output guardrails: "fail_fast" or "collect_all".
	ChainMode string `yaml:"chain_mode,omitempty" json:"chain_mode,omitempty" jsonschema:"title=Chain Mode,enum=fail_fast,enum=collect_all,default=fail_fast"`

	// PII detection/redaction settings.
	PII *PIIGuardrailConfig `yaml:"pii,omitempty" json:"pii,omitempty" jsonschema:"title=PII Detection,description=Detect and redact personally identifiable information"`

	// Content filtering settings.
	Content *ContentGuardrailConfig `yaml:"content,omitempty" json:"content,omitempty" jsonschema:"title=Content Filter,description=Block or warn about harmful content"`
}

// PIIGuardrailConfig configures PII detection and redaction.
type PIIGuardrailConfig struct {
	Enabled          bool   `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=true"`
	DetectEmail      bool   `yaml:"detect_email,omitempty" json:"detect_email,omitempty" jsonschema:"title=Detect Email,default=true"`
	DetectPhone      bool   `yaml:"detect_phone,omitempty" json:"detect_phone,omitempty" jsonschema:"title=Detect Phone,default=true"`
	DetectSSN        bool   `yaml:"detect_ssn,omitempty" json:"detect_ssn,omitempty" jsonschema:"title=Detect SSN,default=true"`
	DetectCreditCard bool   `yaml:"detect_credit_card,omitempty" json:"detect_credit_card,omitempty" jsonschema:"title=Detect Credit Card,default=true"`
	RedactMode       string `yaml:"redact_mode,omitempty" json:"redact_mode,omitempty" jsonschema:"title=Redact Mode,enum=mask,enum=remove,enum=hash,default=mask"`
	Action           string `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=allow,enum=block,enum=modify,enum=warn,default=modify"`
	Severity         string `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"title=Severity,enum=low,enum=medium,enum=high,enum=critical,default=high"`
}

// ContentGuardrailConfig configures content filtering.
type ContentGuardrailConfig struct {
	Enabled         bool     `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=false"`
	BlockedKeywords []string `yaml:"blocked_keywords,omitempty" json:"blocked_keywords,omitempty" jsonschema:"title=Blocked Keywords,description=Case-insensitive keywords to block"`
	BlockedPatterns []string `yaml:"blocked_patterns,omitempty" json:"blocked_patterns,omitempty" jsonschema:"title=Blocked Patterns,description=Regex patterns to block"`
	Action          string   `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=allow,enum=block,enum=warn,default=block"`
	Severity        string   `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"title=Severity,enum=low,enum=medium,enum=high,enum=critical,default=high"`
}

// ToolGuardrailsConfig contains tool guardrail settings.
type ToolGuardrailsConfig struct {
	// ChainMode for tool guardrails: "fail_fast" or "collect_all".
	ChainMode string `yaml:"chain_mode,omitempty" json:"chain_mode,omitempty" jsonschema:"title=Chain Mode,enum=fail_fast,enum=collect_all,default=fail_fast"`

	// Authorization settings.
	Authorization *AuthorizationGuardrailConfig `yaml:"authorization,omitempty" json:"authorization,omitempty" jsonschema:"title=Tool Authorization,description=Control which tools can be called"`
}

// AuthorizationGuardrailConfig configures tool authorization.
type AuthorizationGuardrailConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=false"`
	AllowedTools []string `yaml:"allowed_tools,omitempty" json:"allowed_tools,omitempty" jsonschema:"title=Allowed Tools,description=Whitelist of allowed tools (glob patterns)"`
	BlockedTools []string `yaml:"blocked_tools,omitempty" json:"blocked_tools,omitempty" jsonschema:"title=Blocked Tools,description=Blacklist of blocked tools (glob patterns)"`
	Action       string   `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=allow,enum=block,enum=warn,default=block"`
	Severity     string   `yaml:"severity,omitempty" json:"severity,omitempty" jsonschema:"title=Severity,enum=low,enum=medium,enum=high,enum=critical,default=high"`
}

// ModerationGuardrailConfig configures LLM-powered content moderation.
// This provides semantic understanding beyond regex-based guardrails.
type ModerationGuardrailConfig struct {
	// Enabled controls whether moderation is active.
	Enabled bool `yaml:"enabled" json:"enabled" jsonschema:"title=Enabled,default=false"`

	// Strategy determines which moderation approach to use.
	// Values:
	//   - "none": Disabled
	//   - "openai": OpenAI Moderation API (free)
	//   - "lakera": Lakera Guard API
	//   - "prompt": Custom prompt with any LLM
	Strategy string `yaml:"strategy,omitempty" json:"strategy,omitempty" jsonschema:"title=Strategy,description=Moderation approach,enum=none,enum=openai,enum=lakera,enum=prompt,default=openai"`

	// OpenAI configures the OpenAI Moderation provider.
	OpenAI *OpenAIModerationConfig `yaml:"openai,omitempty" json:"openai,omitempty" jsonschema:"title=OpenAI Config,description=OpenAI Moderation API settings"`

	// Lakera configures the Lakera Guard provider.
	Lakera *LakeraModerationConfig `yaml:"lakera,omitempty" json:"lakera,omitempty" jsonschema:"title=Lakera Config,description=Lakera Guard API settings"`

	// Prompt configures the prompt-based moderation provider.
	Prompt *PromptModerationConfig `yaml:"prompt,omitempty" json:"prompt,omitempty" jsonschema:"title=Prompt Config,description=Custom LLM prompt settings"`

	// Action when content is flagged.
	Action string `yaml:"action,omitempty" json:"action,omitempty" jsonschema:"title=Action,enum=block,enum=warn,default=block"`
}

// OpenAIModerationConfig configures OpenAI's Moderation API.
type OpenAIModerationConfig struct {
	// Model is the moderation model to use.
	// Default: "omni-moderation-latest"
	Model string `yaml:"model,omitempty" json:"model,omitempty" jsonschema:"title=Model,description=Moderation model (omni-moderation-latest or text-moderation-latest),default=omni-moderation-latest"`

	// Threshold is the score above which content is flagged (0-1).
	// Default: 0.8
	Threshold float64 `yaml:"threshold,omitempty" json:"threshold,omitempty" jsonschema:"title=Threshold,description=Score threshold for flagging,minimum=0,maximum=1,default=0.8"`
}

// LakeraModerationConfig configures Lakera Guard v2 API.
type LakeraModerationConfig struct {
	// ProjectID is the Lakera project ID for policy configuration.
	// If empty, uses the default Lakera Guard policy.
	ProjectID string `yaml:"project_id,omitempty" json:"project_id,omitempty" jsonschema:"title=Project ID,description=Lakera project ID for policy configuration"`

	// Breakdown enables detailed detector breakdown in response.
	Breakdown bool `yaml:"breakdown,omitempty" json:"breakdown,omitempty" jsonschema:"title=Breakdown,description=Enable detailed detector breakdown"`

	// Endpoint is the API endpoint URL (optional).
	// Default: https://api.lakera.ai/v2/guard
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty" jsonschema:"title=Endpoint,description=Custom API endpoint URL"`
}

// PromptModerationConfig configures prompt-based moderation.
type PromptModerationConfig struct {
	// LLM references a configured LLM by name.
	// If empty, uses the agent's LLM.
	LLM string `yaml:"llm,omitempty" json:"llm,omitempty" jsonschema:"title=LLM,description=LLM to use for moderation"`

	// Template is the moderation prompt template.
	// Use {input} as placeholder for the content to check.
	Template string `yaml:"template,omitempty" json:"template,omitempty" jsonschema:"title=Template,description=Custom moderation prompt template"`

	// SafeField is the JSON field to check in the response.
	// Default: "safe"
	SafeField string `yaml:"safe_field,omitempty" json:"safe_field,omitempty" jsonschema:"title=Safe Field,default=safe"`
}

// SetDefaults applies default values to the guardrails config.
func (c *GuardrailsConfig) SetDefaults() {
	if c.Input == nil {
		c.Input = &InputGuardrailsConfig{}
	}
	if c.Input.ChainMode == "" {
		c.Input.ChainMode = "fail_fast"
	}

	if c.Output == nil {
		c.Output = &OutputGuardrailsConfig{}
	}
	if c.Output.ChainMode == "" {
		c.Output.ChainMode = "fail_fast"
	}

	if c.Tool == nil {
		c.Tool = &ToolGuardrailsConfig{}
	}
	if c.Tool.ChainMode == "" {
		c.Tool.ChainMode = "fail_fast"
	}
}

// Validate checks the guardrails configuration for errors.
func (c *GuardrailsConfig) Validate() error {
	// Validate chain modes
	validModes := map[string]bool{"fail_fast": true, "collect_all": true, "": true}

	if c.Input != nil && !validModes[c.Input.ChainMode] {
		return fmt.Errorf("invalid input chain_mode: %q", c.Input.ChainMode)
	}
	if c.Output != nil && !validModes[c.Output.ChainMode] {
		return fmt.Errorf("invalid output chain_mode: %q", c.Output.ChainMode)
	}
	if c.Tool != nil && !validModes[c.Tool.ChainMode] {
		return fmt.Errorf("invalid tool chain_mode: %q", c.Tool.ChainMode)
	}

	return nil
}

// DefaultGuardrailsConfig returns sensible default guardrails configuration.
func DefaultGuardrailsConfig() *GuardrailsConfig {
	return &GuardrailsConfig{
		Enabled: true,
		Input: &InputGuardrailsConfig{
			ChainMode: "fail_fast",
			Length: &LengthGuardrailConfig{
				Enabled:   true,
				MinLength: 1,
				MaxLength: 100000,
				Action:    "block",
				Severity:  "medium",
			},
			Injection: &InjectionGuardrailConfig{
				Enabled:       true,
				CaseSensitive: false,
				Action:        "block",
				Severity:      "high",
			},
			Sanitizer: &SanitizerGuardrailConfig{
				Enabled:        true,
				TrimWhitespace: true,
			},
		},
		Output: &OutputGuardrailsConfig{
			ChainMode: "fail_fast",
			PII: &PIIGuardrailConfig{
				Enabled:          true,
				DetectEmail:      true,
				DetectPhone:      true,
				DetectSSN:        true,
				DetectCreditCard: true,
				RedactMode:       "mask",
				Action:           "modify",
				Severity:         "high",
			},
		},
		Tool: &ToolGuardrailsConfig{
			ChainMode: "fail_fast",
		},
	}
}
