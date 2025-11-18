package hector

import (
	"github.com/kadirpekel/hector/pkg/config"
)

// A2ACardBuilder provides a fluent API for building A2A card config
type A2ACardBuilder struct {
	config *config.A2ACardConfig
}

// NewA2ACardBuilder creates a new A2A card builder
func NewA2ACardBuilder(cfg *config.A2ACardConfig) *A2ACardBuilder {
	if cfg == nil {
		cfg = &config.A2ACardConfig{
			InputModes:  make([]string, 0),
			OutputModes: make([]string, 0),
			Skills:      make([]config.A2ASkillConfig, 0),
		}
	}
	if cfg.InputModes == nil {
		cfg.InputModes = make([]string, 0)
	}
	if cfg.OutputModes == nil {
		cfg.OutputModes = make([]string, 0)
	}
	if cfg.Skills == nil {
		cfg.Skills = make([]config.A2ASkillConfig, 0)
	}
	return &A2ACardBuilder{
		config: cfg,
	}
}

// Version sets the A2A card version
func (b *A2ACardBuilder) Version(version string) *A2ACardBuilder {
	b.config.Version = version
	return b
}

// InputModes sets the input modes
func (b *A2ACardBuilder) InputModes(modes []string) *A2ACardBuilder {
	b.config.InputModes = modes
	return b
}

// AddInputMode adds an input mode
func (b *A2ACardBuilder) AddInputMode(mode string) *A2ACardBuilder {
	b.config.InputModes = append(b.config.InputModes, mode)
	return b
}

// OutputModes sets the output modes
func (b *A2ACardBuilder) OutputModes(modes []string) *A2ACardBuilder {
	b.config.OutputModes = modes
	return b
}

// AddOutputMode adds an output mode
func (b *A2ACardBuilder) AddOutputMode(mode string) *A2ACardBuilder {
	b.config.OutputModes = append(b.config.OutputModes, mode)
	return b
}

// Skills sets the skills
func (b *A2ACardBuilder) Skills(skills []config.A2ASkillConfig) *A2ACardBuilder {
	b.config.Skills = skills
	return b
}

// AddSkill adds a skill
func (b *A2ACardBuilder) AddSkill(skill config.A2ASkillConfig) *A2ACardBuilder {
	b.config.Skills = append(b.config.Skills, skill)
	return b
}

// Skill creates a skill builder
func (b *A2ACardBuilder) Skill() *A2ASkillBuilder {
	skill := &config.A2ASkillConfig{}
	b.config.Skills = append(b.config.Skills, *skill)
	return NewA2ASkillBuilder(&b.config.Skills[len(b.config.Skills)-1])
}

// Provider sets the provider configuration
func (b *A2ACardBuilder) Provider(provider *config.A2AProviderConfig) *A2ACardBuilder {
	b.config.Provider = provider
	return b
}

// PreferredTransport sets the preferred transport override
func (b *A2ACardBuilder) PreferredTransport(transport string) *A2ACardBuilder {
	b.config.PreferredTransport = transport
	return b
}

// DocumentationURL sets the documentation URL
func (b *A2ACardBuilder) DocumentationURL(url string) *A2ACardBuilder {
	b.config.DocumentationURL = url
	return b
}

// Build returns the A2A card config
func (b *A2ACardBuilder) Build() *config.A2ACardConfig {
	return b.config
}

// A2ASkillBuilder provides a fluent API for building A2A skill config
type A2ASkillBuilder struct {
	skill *config.A2ASkillConfig
}

// NewA2ASkillBuilder creates a new A2A skill builder
func NewA2ASkillBuilder(skill *config.A2ASkillConfig) *A2ASkillBuilder {
	if skill == nil {
		skill = &config.A2ASkillConfig{}
	}
	return &A2ASkillBuilder{
		skill: skill,
	}
}

// ID sets the skill ID
func (b *A2ASkillBuilder) ID(id string) *A2ASkillBuilder {
	b.skill.ID = id
	return b
}

// Name sets the skill name
func (b *A2ASkillBuilder) Name(name string) *A2ASkillBuilder {
	b.skill.Name = name
	return b
}

// Description sets the skill description
func (b *A2ASkillBuilder) Description(desc string) *A2ASkillBuilder {
	b.skill.Description = desc
	return b
}

// Tags sets the skill tags
func (b *A2ASkillBuilder) Tags(tags []string) *A2ASkillBuilder {
	b.skill.Tags = tags
	return b
}

// AddTag adds a tag
func (b *A2ASkillBuilder) AddTag(tag string) *A2ASkillBuilder {
	b.skill.Tags = append(b.skill.Tags, tag)
	return b
}

// Examples sets the skill examples
func (b *A2ASkillBuilder) Examples(examples []string) *A2ASkillBuilder {
	b.skill.Examples = examples
	return b
}

// AddExample adds an example
func (b *A2ASkillBuilder) AddExample(example string) *A2ASkillBuilder {
	b.skill.Examples = append(b.skill.Examples, example)
	return b
}

// Build returns the skill config
func (b *A2ASkillBuilder) Build() *config.A2ASkillConfig {
	return b.skill
}

