// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"fmt"

	"github.com/kadirpekel/hector/v2/config"
)

// NewValidatorFromConfig creates a TokenValidator from configuration.
// Returns nil if authentication is not enabled.
func NewValidatorFromConfig(cfg *config.AuthConfig) (TokenValidator, error) {
	if cfg == nil || !cfg.IsEnabled() {
		return nil, nil
	}

	// Ensure defaults are applied
	cfg.SetDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid auth config: %w", err)
	}

	// Create JWT validator
	validator, err := NewJWTValidator(JWTValidatorConfig{
		JWKSURL:         cfg.JWKSURL,
		Issuer:          cfg.Issuer,
		Audience:        cfg.Audience,
		RefreshInterval: cfg.RefreshInterval,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT validator: %w", err)
	}

	return validator, nil
}
