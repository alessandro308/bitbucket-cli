package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommentPullRequestValidation(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	tests := []struct {
		name      string
		workspace string
		repoSlug  string
		id        int
		opts      CommentPullRequestOptions
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "missing workspace",
			workspace: "",
			repoSlug:  "repo",
			id:        1,
			opts:      CommentPullRequestOptions{Text: "test"},
			wantErr:   true,
			errMsg:    "workspace and repository slug are required",
		},
		{
			name:      "missing repo slug",
			workspace: "ws",
			repoSlug:  "",
			id:        1,
			opts:      CommentPullRequestOptions{Text: "test"},
			wantErr:   true,
			errMsg:    "workspace and repository slug are required",
		},
		{
			name:      "missing text",
			workspace: "ws",
			repoSlug:  "repo",
			id:        1,
			opts:      CommentPullRequestOptions{Text: ""},
			wantErr:   true,
			errMsg:    "comment text is required",
		},
		{
			name:      "file path without line number",
			workspace: "ws",
			repoSlug:  "repo",
			id:        1,
			opts:      CommentPullRequestOptions{Text: "test", FilePath: "src/main.go", Line: 0},
			wantErr:   true,
			errMsg:    "line number must be positive when file path is specified",
		},
		{
			name:      "invalid line range",
			workspace: "ws",
			repoSlug:  "repo",
			id:        1,
			opts:      CommentPullRequestOptions{Text: "test", FilePath: "src/main.go", Line: 10, LineFrom: 20},
			wantErr:   true,
			errMsg:    "line range start (20) must be less than or equal to end (10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CommentPullRequest(ctx, tt.workspace, tt.repoSlug, tt.id, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CommentPullRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("CommentPullRequest() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestCommentPullRequestPayload(t *testing.T) {
	tests := []struct {
		name         string
		opts         CommentPullRequestOptions
		wantPayload  map[string]any
		wantInline   bool
		wantInlineTo int
	}{
		{
			name: "general comment",
			opts: CommentPullRequestOptions{Text: "Looks good!"},
			wantPayload: map[string]any{
				"content": map[string]any{
					"raw": "Looks good!",
				},
			},
			wantInline: false,
		},
		{
			name: "inline comment on single line",
			opts: CommentPullRequestOptions{
				Text:     "Fix this",
				FilePath: "src/main.go",
				Line:     42,
			},
			wantPayload: map[string]any{
				"content": map[string]any{
					"raw": "Fix this",
				},
				"inline": map[string]any{
					"path": "src/main.go",
					"to":   float64(42),
				},
			},
			wantInline:   true,
			wantInlineTo: 42,
		},
		{
			name: "inline comment on line range",
			opts: CommentPullRequestOptions{
				Text:     "Refactor this block",
				FilePath: "src/helper.go",
				Line:     20,
				LineFrom: 10,
			},
			wantPayload: map[string]any{
				"content": map[string]any{
					"raw": "Refactor this block",
				},
				"inline": map[string]any{
					"path": "src/helper.go",
					"to":   float64(20),
					"from": float64(10),
				},
			},
			wantInline:   true,
			wantInlineTo: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server to capture the request
			var capturedPayload map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&capturedPayload); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}

				// Return a mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(PullRequestComment{
					ID: 123,
					Content: struct {
						Raw    string `json:"raw"`
						Markup string `json:"markup"`
						HTML   string `json:"html"`
					}{
						Raw: tt.opts.Text,
					},
				})
			}))
			defer server.Close()

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			_, err = client.CommentPullRequest(ctx, "ws", "repo", 1, tt.opts)
			if err != nil {
				t.Fatalf("CommentPullRequest() error = %v", err)
			}

			// Verify the payload structure
			if content, ok := capturedPayload["content"].(map[string]any); !ok {
				t.Error("content field missing or invalid")
			} else if raw, ok := content["raw"].(string); !ok || raw != tt.opts.Text {
				t.Errorf("content.raw = %q, want %q", raw, tt.opts.Text)
			}

			// Check inline field
			inline, hasInline := capturedPayload["inline"].(map[string]any)
			if hasInline != tt.wantInline {
				t.Errorf("inline field present = %v, want %v", hasInline, tt.wantInline)
			}

			if tt.wantInline && hasInline {
				if path, ok := inline["path"].(string); !ok || path != tt.opts.FilePath {
					t.Errorf("inline.path = %q, want %q", path, tt.opts.FilePath)
				}
				if to, ok := inline["to"].(float64); !ok || int(to) != tt.wantInlineTo {
					t.Errorf("inline.to = %v, want %d", to, tt.wantInlineTo)
				}
				if tt.opts.LineFrom > 0 {
					if from, ok := inline["from"].(float64); !ok || int(from) != tt.opts.LineFrom {
						t.Errorf("inline.from = %v, want %d", from, tt.opts.LineFrom)
					}
				}
			}
		})
	}
}
