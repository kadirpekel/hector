package rag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// BlobSource implements DataSource for blob storage (local files, S3, GCS, Azure Blob, etc.).
// Uses gocloud.dev/blob to provide unified access to multiple storage backends.
//
// Replaces and extends DirectorySource to support both local and cloud storage:
// - file:///path/to/dir - Local filesystem
// - s3://bucket/prefix?region=us-east-1 - AWS S3
// - gs://bucket/prefix - Google Cloud Storage
// - azblob://container/prefix - Azure Blob Storage
type BlobSource struct {
	bucket      *blob.Bucket
	url         string
	prefix      string
	filter      FileFilter
	maxFileSize int64
	ctx         context.Context
}

// BlobSourceConfig configures a blob storage source.
type BlobSourceConfig struct {
	// URL is the blob storage URL (file://, s3://, gs://, azblob://)
	URL string

	// Prefix filters to blobs with this prefix (optional)
	Prefix string

	// Include patterns for files (same as DirectorySource)
	Include []string

	// Exclude patterns for files (same as DirectorySource)
	Exclude []string

	// MaxFileSize limits file size in bytes
	MaxFileSize int64
}

// NewBlobSource creates a new blob storage data source.
//
// URL examples:
//   - s3://my-bucket/docs?region=us-east-1
//   - gs://my-bucket/docs
//   - azblob://my-container/docs
//
// Environment variables for credentials:
//   - AWS: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
//   - GCS: GOOGLE_APPLICATION_CREDENTIALS
//   - Azure: AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY
func NewBlobSource(ctx context.Context, cfg BlobSourceConfig) (*BlobSource, error) {
	slog.Debug("Creating blob source",
		"url", cfg.URL,
		"prefix", cfg.Prefix,
		"include", cfg.Include,
		"exclude", cfg.Exclude,
		"max_file_size", cfg.MaxFileSize)

	// Open bucket using gocloud URL
	bucket, err := blob.OpenBucket(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob storage at %s: %w", cfg.URL, err)
	}

	// Create filter for blob keys (which are already relative paths)
	var filter FileFilter
	if len(cfg.Include) > 0 || len(cfg.Exclude) > 0 {
		// Use empty base path since blob keys are already relative
		// This prevents filepath.Rel from failing
		filter, err = NewPatternFilter("", cfg.Include, cfg.Exclude)
		if err != nil {
			bucket.Close()
			return nil, fmt.Errorf("failed to create filter: %w", err)
		}
		slog.Debug("Created blob filter",
			"include", cfg.Include,
			"exclude", cfg.Exclude)
	} else {
		slog.Debug("No filter created (no include/exclude patterns)")
	}

	return &BlobSource{
		bucket:      bucket,
		url:         cfg.URL,
		prefix:      cfg.Prefix,
		filter:      filter,
		maxFileSize: cfg.MaxFileSize,
		ctx:         ctx,
	}, nil
}

// NewBlobSourceFromConfig creates a blob source from BlobSourceConfig.
func NewBlobSourceFromConfig(ctx context.Context, cfg BlobSourceConfig) (DataSource, error) {
	return NewBlobSource(ctx, cfg)
}

// Type returns the data source type.
func (bs *BlobSource) Type() string {
	return "blob"
}

// DiscoverDocuments returns channels of discovered documents and errors.
// Documents are discovered asynchronously and sent through the channel.
func (bs *BlobSource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	docChan := make(chan Document, 100)
	errChan := make(chan error, 10)

	go func() {
		defer close(docChan)
		defer close(errChan)

		// List all blobs with the configured prefix
		iter := bs.bucket.List(&blob.ListOptions{
			Prefix: bs.prefix,
		})

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			obj, err := iter.Next(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				select {
				case errChan <- fmt.Errorf("failed to list blobs: %w", err):
				case <-ctx.Done():
					return
				}
				continue
			}

			// Skip directories (blobs ending with /)
			if obj.IsDir {
				continue
			}

			// Skip empty files
			if obj.Size == 0 {
				continue
			}

			// Skip files exceeding max size
			if bs.maxFileSize > 0 && obj.Size > bs.maxFileSize {
				slog.Debug("Skipping file exceeding max size",
					"key", obj.Key,
					"size", obj.Size,
					"max_size", bs.maxFileSize)
				continue
			}

			// Apply filters
			shouldIndex := true
			if bs.filter != nil {
				// For filtering, use the object key as path
				excluded := bs.filter.ShouldExclude(obj.Key)
				included := bs.filter.ShouldInclude(obj.Key)
				slog.Debug("Filter check",
					"key", obj.Key,
					"excluded", excluded,
					"included", included)
				if excluded || !included {
					shouldIndex = false
					slog.Debug("Skipping filtered file",
						"key", obj.Key,
						"should_index", shouldIndex)
					continue // Skip this file entirely if filtered out
				}
			}

			// Read blob content
			data, err := bs.bucket.ReadAll(ctx, obj.Key)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("failed to read blob %s: %w", obj.Key, err):
				case <-ctx.Done():
					return
				}
				continue
			}

			// Create document ID
			// Format: blob:<storage_type>:<key>
			// This avoids issues with colons in URLs
			docID := fmt.Sprintf("blob:%s:%s", bs.getStorageType(), obj.Key)

			// Determine source path (relative to prefix if set)
			sourcePath := obj.Key
			if bs.prefix != "" {
				sourcePath = strings.TrimPrefix(obj.Key, bs.prefix)
				sourcePath = strings.TrimPrefix(sourcePath, "/")
			}

			// Create document
			doc := Document{
				ID:         docID,
				Content:    string(data),
				SourcePath: sourcePath,
				MimeType:   detectMimeType(obj.Key),
				Size:       obj.Size,
				Metadata: map[string]any{
					"key":           obj.Key,
					"source_path":   sourcePath,
					"blob_url":      bs.url,
					"prefix":        bs.prefix,
					"last_modified": obj.ModTime.Unix(),
					"should_index":  shouldIndex,
					"storage_type":  bs.getStorageType(),
				},
			}

			// Add cloud-specific metadata if available
			// Note: ListObject doesn't include ContentType/MD5, only Attributes does
			// We use detectMimeType for content type determination

			select {
			case docChan <- doc:
			case <-ctx.Done():
				return
			}
		}
	}()

	return docChan, errChan
}

// ReadDocument retrieves a specific document by its ID.
// ID format: blob:<storage_type>:<key>
func (bs *BlobSource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	// Parse ID format: blob:<storage_type>:<key>
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 || parts[0] != "blob" {
		return nil, fmt.Errorf("invalid blob document ID format: %s (expected blob:<storage_type>:<key>)", id)
	}

	key := parts[2]

	// Get blob attributes
	attrs, err := bs.bucket.Attributes(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob attributes for %s: %w", key, err)
	}

	// Read blob content
	data, err := bs.bucket.ReadAll(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob %s: %w", key, err)
	}

	// Determine source path
	sourcePath := key
	if bs.prefix != "" {
		sourcePath = strings.TrimPrefix(key, bs.prefix)
		sourcePath = strings.TrimPrefix(sourcePath, "/")
	}

	// Create document
	doc := &Document{
		ID:         id,
		Content:    string(data),
		SourcePath: sourcePath,
		MimeType:   detectMimeType(key),
		Size:       attrs.Size,
		Metadata: map[string]any{
			"key":           key,
			"source_path":   sourcePath,
			"blob_url":      bs.url,
			"prefix":        bs.prefix,
			"last_modified": attrs.ModTime.Unix(),
			"should_index":  true,
			"storage_type":  bs.getStorageType(),
		},
	}

	// Add cloud-specific metadata
	if attrs.ContentType != "" {
		doc.Metadata["content_type"] = attrs.ContentType
	}
	if attrs.MD5 != nil {
		doc.Metadata["md5"] = fmt.Sprintf("%x", attrs.MD5)
	}

	return doc, nil
}

// SupportsIncrementalIndexing returns true as blob sources support incremental indexing.
func (bs *BlobSource) SupportsIncrementalIndexing() bool {
	return true
}

// GetLastModified returns the last modification time for a document.
func (bs *BlobSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	// Parse ID format: blob:<storage_type>:<key>
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 || parts[0] != "blob" {
		return time.Time{}, fmt.Errorf("invalid blob document ID format: %s", id)
	}

	key := parts[2]

	// Get blob attributes
	attrs, err := bs.bucket.Attributes(ctx, key)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get blob attributes: %w", err)
	}

	return attrs.ModTime, nil
}

// Close releases any resources held by the data source.
func (bs *BlobSource) Close() error {
	if bs.bucket != nil {
		return bs.bucket.Close()
	}
	return nil
}

// GetURL returns the blob storage URL.
func (bs *BlobSource) GetURL() string {
	return bs.url
}

// GetPrefix returns the blob prefix filter.
func (bs *BlobSource) GetPrefix() string {
	return bs.prefix
}

// GetFilter returns the file filter.
func (bs *BlobSource) GetFilter() FileFilter {
	return bs.filter
}

// getStorageType infers storage type from URL.
func (bs *BlobSource) getStorageType() string {
	switch {
	case strings.HasPrefix(bs.url, "file://"):
		return "local"
	case strings.HasPrefix(bs.url, "s3://"):
		return "s3"
	case strings.HasPrefix(bs.url, "gs://"):
		return "gcs"
	case strings.HasPrefix(bs.url, "azblob://"):
		return "azure"
	default:
		return "unknown"
	}
}

// DefaultBlobSourceConfig returns default configuration for blob source.
func DefaultBlobSourceConfig(url string) BlobSourceConfig {
	return BlobSourceConfig{
		URL:         url,
		Prefix:      "",
		Include:     nil, // All files by default
		Exclude:     []string{".*", "node_modules", "__pycache__", "vendor", ".git"},
		MaxFileSize: 10 * 1024 * 1024, // 10MB
	}
}

// Ensure BlobSource implements DataSource.
var _ DataSource = (*BlobSource)(nil)
