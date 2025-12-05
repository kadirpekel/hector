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

package auth

import "errors"

// Common authentication errors.
var (
	// ErrUnauthorized is returned when authentication is required but not provided.
	ErrUnauthorized = errors.New("unauthorized: authentication required")

	// ErrForbidden is returned when the user lacks permission.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrInvalidToken is returned when a token cannot be validated.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrMissingClaims is returned when required claims are missing.
	ErrMissingClaims = errors.New("missing required claims")
)
