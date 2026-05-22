// Package ingest takes documents
// from a source directory, chunks them, embeds the chunks, upserts the
// result into a vector.Store, and then moves the originals into a
// "processed" directory. It exposes a long-running Watch entry point
// intended to run as a background goroutine alongside the chat REPL.
//
// The pipeline, in five steps, is the heart of the "indexing side" of
// RAG:
//
//  1. READ   — open a supported text/markdown file from disk.
//  2. CHUNK  — split it into ~1000-byte overlapping windows. We chunk
//     because (a) embedding models have a max input length,
//     (b) we want retrieval to return the *relevant passage*,
//     not a whole document, and (c) shorter chunks produce
//     embeddings that capture local meaning more sharply.
//  3. EMBED  — call the embeddings model once with the full batch of
//     chunks. Each chunk becomes a fixed-dimension float32
//     vector. See internal/llm/embed.go for what an
//     "embedding" actually is.
//  4. DELETE — remove any prior chunks that share this filename, so
//     re-ingesting an edited file leaves no orphans behind.
//  5. UPSERT — insert (or replace) the new chunks in the vector store
//     with the embeddings attached.
//
// At query time (in a later lesson) the user's question will be
// embedded with the SAME model and the store returns the chunks whose
// embeddings sit closest to it. Steps 3 and 5 here are what make that
// lookup possible.

package ingest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Morpa/go-rag/llm"
	"github.com/Morpa/go-rag/vector"
)

const (
	defaultChunkSize    = 1000
	defaultChunkOverlap = 100
)

type Options struct {
	SourceDir    string
	ProcessedDir string
	ChunkSize    int
	ChunkOverlap int
}

func processOne(ctx context.Context, path string, opts Options, embedder llm.Embedder, store vector.Store) error {
	if !supportedFormat(path) {
		return fmt.Errorf("unsupported format: %s", filepath.Ext(path))
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	_, err = processContent(ctx, filepath.Base(path), raw, opts, embedder, store)
	return err
}

func processContent(ctx context.Context, source string, content []byte, opts Options, embedder llm.Embedder, store vector.Store) (int, error) {
	if embedder == nil {
		return 0, errors.New("embedder is required")
	}

	if store == nil {
		return 0, errors.New("vector store is required")
	}

	base := filepath.Base(source)
	if !supportedFormat(base) {
		return 0, fmt.Errorf("unsupported format: %s", filepath.Ext(base))
	}

	size := opts.ChunkSize
	if size <= 0 {
		size = defaultChunkSize
	}

	overlap := opts.ChunkOverlap
	if overlap <= 0 {
		overlap = defaultChunkOverlap
	}

	text := strings.TrimSpace(string(content))
	if text == "" {
		return 0, errors.New("file is empty")
	}

	chunks := chunk(text, size, overlap)
	if len(chunks) == 0 {
		return 0, errors.New("no chunks produced")
	}

	vectors, err := embedder.Embed(ctx, chunks)
	if err != nil {
		return 0, fmt.Errorf("embed: %w", err)
	}

	if len(vectors) != len(chunks) {
		return 0, fmt.Errorf("embed: got %d vectors for %d chunks", len(vectors), len(chunks))
	}

	if err := store.DeleteBySource(ctx, base); err != nil {
		return 0, fmt.Errorf("clear previous chunks: %w", err)
	}

	ingestedAt := time.Now().UTC().Format(time.RFC3339)
	docs := make([]vector.Document, len(chunks))
	for i, c := range chunks {
		docs[i] = vector.Document{
			ID:      fmt.Sprintf("%s#%d", base, i),
			Content: c,
			Metadata: map[string]string{
				"source":      base,
				"chunk_index": strconv.Itoa(i),
				"chunks":      strconv.Itoa(len(chunks)),
				"ingested_at": ingestedAt,
			},
			Embedding: vectors[i],
		}
	}

	if err := store.Upsert(ctx, docs); err != nil {
		return 0, err
	}

	return len(chunks), nil
}

func supportedFormat(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md", ".markdown":
		return true
	}
	return false
}
