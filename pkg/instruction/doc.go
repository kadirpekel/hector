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

// Package instruction provides instruction templating for Hector v2 agents.
//
// This package implements adk-go compatible instruction templating, allowing
// agent instructions to contain dynamic placeholders that are resolved at runtime.
//
// # Placeholder Syntax
//
// Placeholders use curly braces and support several forms:
//
//	{variable}           - Session state variable
//	{app:variable}       - App-scoped state (shared across all users/sessions)
//	{user:variable}      - User-scoped state (shared across sessions for a user)
//	{temp:variable}      - Temporary state (discarded after invocation)
//	{artifact.filename}  - Artifact text content
//	{variable?}          - Optional (empty string if not found, no error)
//
// # Usage
//
// Basic usage with InjectState:
//
//	template := "Hello {user_name}, you are working on {app:project_name}."
//	resolved, err := instruction.InjectState(ctx, template)
//	if err != nil {
//	    return err
//	}
//	// resolved: "Hello Alice, you are working on MyProject."
//
// Using the Template type:
//
//	tmpl := instruction.New("Task: {task}\nContext: {artifact.context?}")
//	resolved, err := tmpl.Render(ctx)
//
// # Integration with LLM Agents
//
// The instruction package is used by llmagent to resolve instruction templates
// before sending to the LLM:
//
//	agent, _ := llmagent.New(llmagent.Config{
//	    Name: "assistant",
//	    Instruction: `
//	        You are helping {user_name?} with {task}.
//
//	        Project context:
//	        {artifact.project_context?}
//	    `,
//	})
//
// # Error Handling
//
// Required placeholders (without ?) return an error if not found.
// Optional placeholders (with ?) return an empty string if not found.
// Invalid placeholder names (not valid identifiers) are left as-is.
package instruction
