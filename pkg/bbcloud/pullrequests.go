package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// PullRequest models a Bitbucket Cloud pull request.
type PullRequest struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
		Repository struct {
			Slug string `json:"slug"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			Slug string `json:"slug"`
		} `json:"repository"`
	} `json:"destination"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Summary struct {
		Raw string `json:"raw"`
	} `json:"summary"`
}

// PullRequestListOptions configure PR listings.
type PullRequestListOptions struct {
	State string
	Limit int
	Mine  string
}

type pullRequestListPage struct {
	Values []PullRequest `json:"values"`
	Next   string        `json:"next"`
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(ctx context.Context, workspace, repoSlug string, opts PullRequestListOptions) ([]PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 20
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))
	if state := strings.TrimSpace(opts.State); state != "" && !strings.EqualFold(state, "all") {
		params = append(params, "state="+url.QueryEscape(strings.ToUpper(state)))
	}
	if opts.Mine != "" {
		params = append(params, "q="+url.QueryEscape(fmt.Sprintf("author.username=\"%s\"", opts.Mine)))
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	var prs []PullRequest
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page pullRequestListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		prs = append(prs, page.Values...)

		if opts.Limit > 0 && len(prs) >= opts.Limit {
			prs = prs[:opts.Limit]
			break
		}

		if page.Next == "" {
			break
		}
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return prs, nil
}

// GetPullRequest fetches a pull request by ID.
func (c *Client) GetPullRequest(ctx context.Context, workspace, repoSlug string, id int) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// CreatePullRequestInput configures PR creation.
type CreatePullRequestInput struct {
	Title       string
	Description string
	Source      string
	Destination string
	CloseSource bool
	Reviewers   []string
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, workspace, repoSlug string, input CreatePullRequestInput) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.Destination) == "" {
		return nil, fmt.Errorf("source and destination branches are required")
	}

	body := map[string]any{
		"title":               input.Title,
		"close_source_branch": input.CloseSource,
		"source": map[string]any{
			"branch": map[string]string{"name": input.Source},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": input.Destination},
		},
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if len(input.Reviewers) > 0 {
		var reviewers []map[string]string
		for _, reviewer := range input.Reviewers {
			reviewers = append(reviewers, map[string]string{"username": reviewer})
		}
		body["reviewers"] = reviewers
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// UpdatePullRequestInput configures PR updates. Use pointers to distinguish
// between "not set" and "set to empty string" for clearing fields.
type UpdatePullRequestInput struct {
	Title       *string
	Description *string
}

// UpdatePullRequest updates an existing pull request's title and/or description.
func (c *Client) UpdatePullRequest(ctx context.Context, workspace, repoSlug string, id int, input UpdatePullRequestInput) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	body := make(map[string]any)
	if input.Title != nil {
		body["title"] = *input.Title
	}
	if input.Description != nil {
		body["description"] = *input.Description
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("at least one field (title or description) must be provided")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)

	req, err := c.http.NewRequest(ctx, "PUT", path, body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// PullRequestComment represents a comment on a pull request.
type PullRequestComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
	} `json:"content"`
	User      *Account `json:"user"`
	CreatedOn string   `json:"created_on"`
	UpdatedOn string   `json:"updated_on"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// CommentPullRequestOptions configures PR comment creation.
type CommentPullRequestOptions struct {
	Text     string
	FilePath string // Optional: file path for inline comment
	Line     int    // Optional: line number for inline comment (requires FilePath)
	LineFrom int    // Optional: starting line for range comment (requires FilePath and Line)
}

// CommentPullRequest creates a comment on a pull request.
// For inline comments on specific file lines, set FilePath and Line in the options.
func (c *Client) CommentPullRequest(ctx context.Context, workspace, repoSlug string, id int, opts CommentPullRequestOptions) (*PullRequestComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(opts.Text) == "" {
		return nil, fmt.Errorf("comment text is required")
	}

	payload := map[string]any{
		"content": map[string]any{
			"raw": opts.Text,
		},
	}

	// Add inline comment location if file and line are specified
	if opts.FilePath != "" {
		if opts.Line <= 0 {
			return nil, fmt.Errorf("line number must be positive when file path is specified")
		}

		inline := map[string]any{
			"path": opts.FilePath,
			"to":   opts.Line,
		}

		// Add line range if LineFrom is specified
		if opts.LineFrom > 0 {
			if opts.LineFrom > opts.Line {
				return nil, fmt.Errorf("line range start (%d) must be less than or equal to end (%d)", opts.LineFrom, opts.Line)
			}
			inline["from"] = opts.LineFrom
		}

		payload["inline"] = inline
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)

	req, err := c.http.NewRequest(ctx, "POST", path, payload)
	if err != nil {
		return nil, err
	}

	var comment PullRequestComment
	if err := c.http.Do(req, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}
