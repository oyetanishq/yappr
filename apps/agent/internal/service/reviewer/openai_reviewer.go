package reviewer

import (
	"context"
	"fmt"
	"strings"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// ── Prompts ───────────────────────────────────────────────────────────────────

// systemPromptSummary instructs the model to produce a PR summary and per-file analysis.
var systemPromptSummary = "You are a friendly assistant reviewing a pull request.\n\n" +
	"Your task is TWO things:\n" +
	"1. Write a very simple, non-technical summary of what this PR changed (2-4 bullet points).\n" +
	"   Explain it in plain English so that a 10-year-old could understand it. Do not use complex architecture terms.\n\n" +
	"2. Create a Markdown table detailing the technical changes for each changed file.\n" +
	"   The table should have three columns: 'File', 'Change Type', and 'Technical Detail'.\n" +
	"   Keep the technical details brief (1-2 short sentences).\n\n" +
	"Format output as valid GitHub-Flavored Markdown.\n\n" +
	"Output format:\n" +
	"## PR Summary\n" +
	"- bullet 1\n" +
	"- bullet 2\n" +
	"...\n\n" +
	"## Changed Files\n\n" +
	"| File | Change Type | Technical Detail |\n" +
	"|---|---|---|\n" +
	"| `path/to/file.go` | Added / Modified / Removed | Brief explanation of technical change |\n"

// systemPromptDiagram instructs the model to produce a Mermaid flowchart of the new execution flow.
var systemPromptDiagram = "You are a software architect who generates precise Mermaid flowchart diagrams.\n\n" +
	"Given a set of code changes, produce ONE Mermaid flowchart that shows the NEW execution flow introduced by this PR.\n\n" +
	"Rules:\n" +
	"- Ensure the Mermaid code is perfectly valid and renders without any errors.\n" +
	"- Avoid special characters in node names that might break Mermaid syntax (use quotes if necessary like id[\"Label(text)\"]).\n" +
	"- Focus on: function call chains, data transformations, API boundaries, decision points, error paths\n" +
	"- Use meaningful node labels (function names, not generic 'step 1')\n" +
	"- Show error paths with dashed arrows\n" +
	"- Keep it to maximum 20 nodes\n" +
	"- Output ONLY the raw mermaid code block — nothing before or after it\n\n" +
	"Example output format:\n" +
	"```mermaid\n" +
	"flowchart TD\n" +
	"    A[APIHandler] --> B[validateInput]\n" +
	"    B -->|valid| C[processPayment]\n" +
	"    B -->|invalid| D[return 400]\n" +
	"    C --> E{PaymentGateway}\n" +
	"    E -->|success| F[createOrder]\n" +
	"    E -->|fail| G[rollback]\n" +
	"```"

// systemPromptBugDetection instructs the model to perform deep, evidence-based bug detection.
var systemPromptBugDetection = "You are an elite security and reliability engineer performing a deep code review.\n\n" +
	"Your ONLY mission: find REAL bugs, potential errors, edge cases, and security vulnerabilities.\n\n" +
	"You have been given:\n" +
	"1. The exact code diff with context\n" +
	"2. Full content of changed files\n" +
	"3. All call sites in the blast radius (files that call the changed functions)\n" +
	"4. Related test files\n\n" +
	"## MANDATORY RULES:\n" +
	"- ONLY report issues you can prove from the code shown — no speculation\n" +
	"- Cite EXACT file path + line number for every finding\n" +
	"- Quote the exact problematic code line(s) as evidence\n" +
	"- Provide a specific code fix, not vague advice\n" +
	"- Do NOT report: style issues, naming conventions, missing comments, or minor improvements\n" +
	"- Each finding must have a severity: CRITICAL / HIGH / MEDIUM / LOW\n\n" +
	"## Bug Categories to Systematically Check:\n\n" +
	"**Nil/Null Safety**\n" +
	"- Unchecked nil returns before method calls\n" +
	"- Type assertions without the comma-ok pattern: x := v.(Type) instead of x, ok := v.(Type)\n" +
	"- Map access without existence check on used result\n" +
	"- Pointer dereference without nil guard\n\n" +
	"**Error Handling**\n" +
	"- Errors silently discarded with _ = err or no check\n" +
	"- Wrong error type returned (wrapping vs sentinel)\n" +
	"- Error returned from goroutine that no one reads\n" +
	"- Context cancellation not propagated (ctx passed to function but unused)\n\n" +
	"**Concurrency & Race Conditions**\n" +
	"- Shared map/slice written from multiple goroutines without mutex\n" +
	"- WaitGroup counter decremented before all goroutines finish\n" +
	"- Channel operations that can deadlock\n" +
	"- Goroutine leak (goroutine started, context cancelled, but goroutine never exits)\n\n" +
	"**Resource Leaks**\n" +
	"- http.Response.Body not closed (defer resp.Body.Close() missing)\n" +
	"- Database rows not closed\n" +
	"- File handles not closed\n" +
	"- Ticker/Timer not stopped\n\n" +
	"**Integer & Bounds**\n" +
	"- Integer overflow: arithmetic on user-controlled int without bounds check\n" +
	"- Slice/array index used without length check\n" +
	"- Off-by-one in loop bounds\n\n" +
	"**API Contract Breakage (from blast radius)**\n" +
	"- Callers in blast-radius files using the OLD function signature\n" +
	"- Interface implementors that now violate the updated interface\n" +
	"- Return type change that breaks callers\n\n" +
	"**Security**\n" +
	"- SQL injection via string concatenation in a query\n" +
	"- Command injection via exec.Command with user input\n" +
	"- Path traversal via unchecked filepath.Join with user input\n" +
	"- Hardcoded secrets or tokens in code\n" +
	"- Auth/permission check removed by refactoring\n" +
	"- Unvalidated redirect URLs\n\n" +
	"**Logic Errors**\n" +
	"- Dead/unreachable code after a return/panic/break\n" +
	"- Condition that is always true or always false\n" +
	"- Wrong comparison operator (= vs ==, < vs <=)\n" +
	"- Missing default case in switch that handles exhaustive enum\n" +
	"- Incorrect error sentinel comparison (== vs errors.Is())\n\n" +
	"## Output Format:\n" +
	"For each finding:\n\n" +
	"---\n" +
	"### [SEVERITY] Short descriptive title\n\n" +
	"**Location**: `path/to/file.go:line_number`\n\n" +
	"**Evidence**:\n" +
	"```language\n" +
	"exact problematic code\n" +
	"```\n\n" +
	"**Why it's a bug**: Clear explanation of the failure scenario and when it would trigger.\n\n" +
	"**Fix**:\n" +
	"```language\n" +
	"corrected code with the fix applied\n" +
	"```\n\n" +
	"---\n\n" +
	"If you find NO issues, say: '✅ No bugs or edge cases found in this PR.' and stop."

// ── OpenAIReviewer ────────────────────────────────────────────────────────────

// OpenAIReviewer executes 3 passes of LLM calls to produce a full code review using OpenAI's API.
type OpenAIReviewer struct {
	client *openai.Client
	log    *zap.Logger
	cfg    *config.Config
}

// NewOpenAIReviewer creates a OpenAIReviewer using the provided API key.
func NewOpenAIReviewer(cfg *config.Config, log *zap.Logger) *OpenAIReviewer {
	config := openai.DefaultConfig(cfg.OpenAI.APIKey)
	config.BaseURL = cfg.OpenAI.BaseURL

	c := openai.NewClientWithConfig(config)
	return &OpenAIReviewer{
		client: c,
		log:    log,
		cfg:    cfg,
	}
}

// Review runs all three passes and returns a ReviewResult.
func (r *OpenAIReviewer) Review(ctx context.Context, rc *ReviewContext) (*ReviewResult, error) {
	result := &ReviewResult{}

	// ── Pass A: PR Summary + File Changes ──────────
	r.log.Info("reviewer: pass A — summary + file changes")
	summaryAndFiles, err := r.passA(ctx, rc)
	if err != nil {
		return nil, fmt.Errorf("pass A failed: %w", err)
	}

	result.Summary, result.FileChanges = splitSummaryAndFiles(summaryAndFiles)
	r.log.Info("reviewer: pass A complete")

	// ── Pass B: Architecture Flow Diagram ──────────────
	r.log.Info("reviewer: pass B — architecture diagram")
	diagram, err := r.passB(ctx, rc)
	if err != nil {
		r.log.Warn("reviewer: pass B failed, skipping diagram", zap.Error(err))
		result.FlowDiagram = ""
	} else {
		result.FlowDiagram = diagram
		r.log.Info("reviewer: pass B complete")
	}

	// ── Pass C: Bug Detection ──────────────────────
	r.log.Info("reviewer: pass C — deep bug detection")
	bugReport, err := r.passC(ctx, rc)
	if err != nil {
		return nil, fmt.Errorf("pass C failed: %w", err)
	}
	result.BugReport = bugReport
	r.log.Info("reviewer: pass C complete")

	return result, nil
}

// ── Pass A ────────────────────────────────────────────────────────────────────

func (r *OpenAIReviewer) passA(ctx context.Context, rc *ReviewContext) (string, error) {
	userMsg := fmt.Sprintf(`%s

%s

%s

Analyze this pull request and produce:
1. A bullet-point PR summary (what this PR accomplishes)
2. Per-file change analysis (what changed in each file and why it matters)`,
		rc.PRMeta,
		rc.RepoSummary,
		rc.ChangedFilesBlock,
	)

	return r.CallLLM(ctx, systemPromptSummary, userMsg, r.cfg.OpenAI.BaseModel)
}

// ── Pass B ────────────────────────────────────────────────────────────────────

func (r *OpenAIReviewer) passB(ctx context.Context, rc *ReviewContext) (string, error) {
	userMsg := fmt.Sprintf(`%s

Here are the changed files:
%s

Generate a Mermaid flowchart showing the NEW execution flow introduced by this PR.
Focus on the most architecturally significant change. Output ONLY the mermaid code block.`,
		rc.PRMeta,
		truncateToChars(rc.ChangedFilesBlock, 15_000),
	)

	return r.CallLLM(ctx, systemPromptDiagram, userMsg, r.cfg.OpenAI.BaseModel)
}

// ── Pass C ────────────────────────────────────────────────────────────────────

func (r *OpenAIReviewer) passC(ctx context.Context, rc *ReviewContext) (string, error) {
	userMsg := fmt.Sprintf(`## PR Metadata
%s

## Repository Context
%s

## Changed Code (with diffs)
%s

Perform a systematic deep analysis. Check EVERY category listed in the system prompt.
Find every real bug, potential error, dangerous edge case, and security issue.
Be specific — cite exact file:line for each finding.`,
		rc.PRMeta,
		rc.RepoSummary,
		rc.ChangedFilesBlock,
	)

	return r.CallLLM(ctx, systemPromptBugDetection, userMsg, r.cfg.OpenAI.BaseModel)
}

// ── API call helpers ──────────────────────────────────────────────────────────

func (r *OpenAIReviewer) CallLLM(ctx context.Context, systemPrompt, userMsg, model string) (string, error) {
	resp, err := r.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userMsg,
				},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("openai API (OpenAI): %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// ── Output helpers ────────────────────────────────────────────────────────────

// splitSummaryAndFiles splits Pass A output at the "## Changed Files" marker.
func splitSummaryAndFiles(text string) (summary, files string) {
	marker := "## Changed Files"
	idx := strings.Index(text, marker)
	if idx < 0 {
		return strings.TrimSpace(text), ""
	}
	return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx:])
}

// truncateToChars truncates a string to maxChars, preserving whole lines.
func truncateToChars(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	// Find last newline before limit
	cut := strings.LastIndex(s[:maxChars], "\n")
	if cut < 0 {
		cut = maxChars
	}
	return s[:cut] + "\n\n... [context truncated to fit token budget]"
}
