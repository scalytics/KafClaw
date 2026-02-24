package memory

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/KafClaw/KafClaw/internal/provider"
)

// fakeEmbedder returns a fixed vector for any input.
type fakeEmbedder struct {
	vector []float32
}

func (f *fakeEmbedder) Embed(ctx context.Context, req *provider.EmbeddingRequest) (*provider.EmbeddingResponse, error) {
	return &provider.EmbeddingResponse{Vector: f.vector}, nil
}

type failingEmbedder struct{}

func (f *failingEmbedder) Embed(ctx context.Context, req *provider.EmbeddingRequest) (*provider.EmbeddingResponse, error) {
	return nil, errors.New("embed failed")
}

type fakeVectorStore struct {
	upsertErr error
	searchErr error
	results   []Result
}

func (f *fakeVectorStore) Upsert(ctx context.Context, id string, vector []float32, payload map[string]interface{}) error {
	return f.upsertErr
}
func (f *fakeVectorStore) Search(ctx context.Context, vector []float32, limit int) ([]Result, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.results, nil
}
func (f *fakeVectorStore) EnsureCollection(ctx context.Context) error { return nil }

type fakeTextStore struct {
	fakeVectorStore
	upsertTextErr error
	searchTextErr error
	textResults   []Result
}

func (f *fakeTextStore) UpsertText(ctx context.Context, id string, payload map[string]interface{}) error {
	return f.upsertTextErr
}
func (f *fakeTextStore) SearchText(ctx context.Context, query string, limit int) ([]Result, error) {
	if f.searchTextErr != nil {
		return nil, f.searchTextErr
	}
	return f.textResults, nil
}

func TestMemoryService_StoreAndSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewSQLiteVecStore(db, 3)
	emb := &fakeEmbedder{vector: []float32{1, 0, 0}}
	svc := NewMemoryService(store, emb)
	ctx := context.Background()

	id, err := svc.Store(ctx, "I prefer short answers", "user", "prefs")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	chunks, err := svc.Search(ctx, "preferences", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "I prefer short answers" {
		t.Errorf("unexpected content: %q", chunks[0].Content)
	}
	if chunks[0].Source != "user" {
		t.Errorf("unexpected source: %q", chunks[0].Source)
	}
}

func TestMemoryService_NilEmbedder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteVecStore(db, 3)
	svc := NewMemoryService(store, nil)
	ctx := context.Background()

	// Store should still persist via text-only fallback
	id, err := svc.Store(ctx, "something", "user", "")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Errorf("expected non-empty ID with text fallback")
	}

	// Search should return lexical fallback result
	chunks, err := svc.Search(ctx, "something", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 fallback chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "something" {
		t.Fatalf("unexpected fallback content: %q", chunks[0].Content)
	}
}

func TestMemoryService_EmbedErrorFallsBackToTextSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store := NewSQLiteVecStore(db, 3)
	svc := NewMemoryService(store, &failingEmbedder{})
	ctx := context.Background()

	// Pre-store via text-only path first.
	ts := NewMemoryService(store, nil)
	if _, err := ts.Store(ctx, "router runbook", "user", "ops"); err != nil {
		t.Fatal(err)
	}

	chunks, err := svc.Search(ctx, "router", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected lexical fallback result, got %d", len(chunks))
	}
}

func TestMemoryService_StoreErrorPaths(t *testing.T) {
	ctx := context.Background()

	svc := NewMemoryService(&fakeVectorStore{}, nil)
	id, err := svc.Store(ctx, "a", "user", "")
	if err != nil || id != "" {
		t.Fatalf("expected graceful no-op for nil embedder + non-text store, got id=%q err=%v", id, err)
	}

	svc = NewMemoryService(&fakeTextStore{upsertTextErr: errors.New("x")}, nil)
	if _, err := svc.Store(ctx, "a", "user", ""); err == nil {
		t.Fatal("expected upsert text error")
	}

	svc = NewMemoryService(&fakeVectorStore{upsertErr: errors.New("x")}, &fakeEmbedder{vector: []float32{1}})
	if _, err := svc.Store(ctx, "a", "user", ""); err == nil {
		t.Fatal("expected vector upsert error")
	}

	svc = NewMemoryService(&fakeVectorStore{}, &failingEmbedder{})
	if _, err := svc.Store(ctx, "a", "user", ""); err == nil {
		t.Fatal("expected embed error")
	}
}

func TestMemoryService_SearchFallbackBranches(t *testing.T) {
	ctx := context.Background()

	// No text-capable fallback path.
	svc := NewMemoryService(&fakeVectorStore{}, nil)
	chunks, err := svc.Search(ctx, "q", 0)
	if err != nil {
		t.Fatal(err)
	}
	if chunks != nil {
		t.Fatalf("expected nil without fallback store, got %+v", chunks)
	}

	// Text fallback error path.
	svc = NewMemoryService(&fakeTextStore{searchTextErr: errors.New("boom")}, nil)
	if _, err := svc.Search(ctx, "q", 5); err == nil {
		t.Fatal("expected text fallback error")
	}

	// Vector search error -> text fallback.
	svc = NewMemoryService(&fakeTextStore{
		fakeVectorStore: fakeVectorStore{searchErr: errors.New("vector failed")},
		textResults: []Result{{
			ID:    "x",
			Score: 1,
			Payload: map[string]interface{}{
				"content": "fallback",
				"source":  "user",
				"tags":    "",
			},
		}},
	}, &fakeEmbedder{vector: []float32{1}})
	chunks, err = svc.Search(ctx, "q", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].Content != "fallback" {
		t.Fatalf("expected vector-error fallback result, got %+v", chunks)
	}
}

func TestMemoryService_SearchBySource(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryService(&fakeTextStore{
		textResults: []Result{
			{ID: "1", Score: 1, Payload: map[string]interface{}{"content": "a", "source": "conversation:slack", "tags": ""}},
			{ID: "2", Score: 1, Payload: map[string]interface{}{"content": "b", "source": "tool:read_file", "tags": ""}},
			{ID: "3", Score: 1, Payload: map[string]interface{}{"content": "c", "source": "conversation:wa", "tags": ""}},
		},
	}, nil)

	results, err := svc.SearchBySource(ctx, "q", "conversation:", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected filtered limit 2, got %d", len(results))
	}
	for _, r := range results {
		if !strings.HasPrefix(r.Source, "conversation:") {
			t.Fatalf("unexpected filtered source: %s", r.Source)
		}
	}

	svc = NewMemoryService(&fakeTextStore{searchTextErr: errors.New("x")}, nil)
	if _, err := svc.SearchBySource(ctx, "q", "conversation:", 2); err == nil {
		t.Fatal("expected SearchBySource to surface search error")
	}
}

func TestChunkByHeaders(t *testing.T) {
	content := `# Title

Some intro.

## Section A

Content A line 1.
Content A line 2.

## Section B

Content B.
`
	chunks := ChunkByHeaders(content, "test.md")
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks (preamble + 2 sections), got %d", len(chunks))
	}
	// Preamble chunk uses sourceName as heading
	if chunks[0].Heading != "test.md" {
		t.Errorf("expected preamble heading 'test.md', got %q", chunks[0].Heading)
	}
	if chunks[1].Heading != "Section A" {
		t.Errorf("expected heading 'Section A', got %q", chunks[1].Heading)
	}
	if chunks[2].Heading != "Section B" {
		t.Errorf("expected heading 'Section B', got %q", chunks[2].Heading)
	}
}

func TestChunkByHeaders_NoHeaders(t *testing.T) {
	content := "Just a plain paragraph.\nAnother line."
	chunks := ChunkByHeaders(content, "notes.md")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (whole content), got %d", len(chunks))
	}
	if chunks[0].Heading != "notes.md" {
		t.Errorf("expected heading 'notes.md', got %q", chunks[0].Heading)
	}
}

func TestChunkByHeaders_Empty(t *testing.T) {
	chunks := ChunkByHeaders("", "empty.md")
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestChunkID_Deterministic(t *testing.T) {
	id1 := chunkID("user", "hello")
	id2 := chunkID("user", "hello")
	if id1 != id2 {
		t.Errorf("expected same ID, got %q and %q", id1, id2)
	}

	id3 := chunkID("user", "world")
	if id1 == id3 {
		t.Error("expected different IDs for different content")
	}
}
