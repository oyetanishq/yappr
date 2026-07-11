package model

import "time"

// RunStatus is the lifecycle state of a single PR review run.
type RunStatus string

const (
	// RunStatusProcessing means the review pipeline has started but not yet finished.
	RunStatusProcessing RunStatus = "processing"
	// RunStatusCompleted means the review finished and the result was posted.
	RunStatusCompleted RunStatus = "completed"
	// RunStatusFailed means the pipeline errored (fetch or AI step).
	RunStatusFailed RunStatus = "failed"
	// RunStatusLimitReached means the review was skipped because the user hit
	// their monthly free-tier PR cap.
	RunStatusLimitReached RunStatus = "limit_reached"
)

// PRRun records a single AI code-review run for a pull request. One document is
// created per reviewed PR (on the "opened" event) and updated as the pipeline
// progresses. It is the source of truth for the dashboard's PR review history.
type PRRun struct {
	ID             string      `bson:"_id"             json:"id"`
	UserID         string      `bson:"user_id"         json:"-"` // never leak to the client
	InstallationID int64       `bson:"installation_id" json:"installation_id"`
	RepoFullName   string      `bson:"repo_full_name"  json:"repo_full_name"` // "owner/repo"
	PRNumber       int         `bson:"pr_number"       json:"pr_number"`
	PRTitle        string      `bson:"pr_title"        json:"pr_title"`
	PRURL          string      `bson:"pr_url"          json:"pr_url"` // GitHub PR html_url
	Author         string      `bson:"author"          json:"author"`
	HeadSHA        string      `bson:"head_sha"        json:"head_sha"`
	BaseSHA        string      `bson:"base_sha"        json:"base_sha"`
	Personality    Personality `bson:"personality"     json:"personality"`
	Status         RunStatus   `bson:"status"          json:"status"`
	Error          string      `bson:"error,omitempty" json:"error,omitempty"`

	// Diff stats (populated on completion, from ReviewContext).
	FilesChanged int `bson:"files_changed" json:"files_changed"`
	Additions    int `bson:"additions"     json:"additions"`
	Deletions    int `bson:"deletions"     json:"deletions"`

	// Full review content (populated on completion) — the three LLM passes.
	Summary     string `bson:"summary,omitempty"      json:"summary,omitempty"`
	FileChanges string `bson:"file_changes,omitempty" json:"file_changes,omitempty"`
	FlowDiagram string `bson:"flow_diagram,omitempty" json:"flow_diagram,omitempty"`
	BugReport   string `bson:"bug_report,omitempty"   json:"bug_report,omitempty"`

	CreatedAt   time.Time  `bson:"created_at"             json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"             json:"updated_at"`
	CompletedAt *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}
