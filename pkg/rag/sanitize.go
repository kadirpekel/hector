// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import "strings"

// sanitizeInput removes or escapes potential prompt injection patterns from user input.
// This prevents malicious queries from manipulating the LLM's behavior.
//
// Direct port from legacy pkg/context/sanitize.go
func sanitizeInput(input string) string {
	// Remove common prompt injection patterns
	sanitized := input

	// Remove system role indicators that could confuse the LLM
	sanitized = strings.ReplaceAll(sanitized, "SYSTEM:", "")
	sanitized = strings.ReplaceAll(sanitized, "System:", "")
	sanitized = strings.ReplaceAll(sanitized, "system:", "")
	sanitized = strings.ReplaceAll(sanitized, "ASSISTANT:", "")
	sanitized = strings.ReplaceAll(sanitized, "Assistant:", "")
	sanitized = strings.ReplaceAll(sanitized, "assistant:", "")
	sanitized = strings.ReplaceAll(sanitized, "USER:", "")
	sanitized = strings.ReplaceAll(sanitized, "User:", "")
	sanitized = strings.ReplaceAll(sanitized, "user:", "")

	// Remove instruction override attempts
	sanitized = strings.ReplaceAll(sanitized, "Ignore previous instructions", "")
	sanitized = strings.ReplaceAll(sanitized, "ignore previous instructions", "")
	sanitized = strings.ReplaceAll(sanitized, "Ignore all previous", "")
	sanitized = strings.ReplaceAll(sanitized, "ignore all previous", "")
	sanitized = strings.ReplaceAll(sanitized, "Disregard previous", "")
	sanitized = strings.ReplaceAll(sanitized, "disregard previous", "")

	// Remove common delimiter attacks (trying to break out of the prompt structure)
	sanitized = strings.ReplaceAll(sanitized, "---", "")
	sanitized = strings.ReplaceAll(sanitized, "===", "")
	sanitized = strings.ReplaceAll(sanitized, "***", "")

	// Escape backticks that could be used for code injection or markdown manipulation
	sanitized = strings.ReplaceAll(sanitized, "```", "")

	// Remove excessive whitespace that could be used for obfuscation
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}
