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

// Package embedder provides text embedding services for semantic search.
//
// Ported from legacy pkg/embedders for use in v2.
package embedder

import (
	"context"
)

// Embedder produces vector embeddings from text.
//
// Embeddings are used by IndexService for semantic similarity search.
// Different providers (OpenAI, Ollama) implement this interface.
type Embedder interface {
	// Embed converts text to a vector embedding.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch converts multiple texts to vector embeddings.
	// More efficient than calling Embed multiple times.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the embedding vector dimension.
	Dimension() int

	// Model returns the model name being used.
	Model() string

	// Close releases any resources held by the embedder.
	Close() error
}
