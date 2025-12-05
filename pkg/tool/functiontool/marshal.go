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

package functiontool

import (
	"encoding/json"
	"fmt"
)

// mapToStruct converts a map[string]any to a typed struct.
// This uses JSON marshaling/unmarshaling to handle type conversion properly.
func mapToStruct(m map[string]any, target any) error {
	if m == nil {
		return nil
	}

	// Marshal map to JSON
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	// Unmarshal JSON to target struct
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal args: %w", err)
	}

	return nil
}
