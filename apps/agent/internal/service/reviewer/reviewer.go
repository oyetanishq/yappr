package reviewer

import (
	"context"
	"fmt"
	"strings"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// ── Personality tone modifiers ────────────────────────────────────────────────

// personalityTone returns a tone-modifier string prepended to every system prompt
// when the given personality is selected. The modifier shapes the reviewer's voice
// without replacing the technical instructions.
func personalityTone(p model.Personality) string {
	switch p {
	case model.PersonalityBestie:
		return "## 🧸 Your Reviewer Personality: The Bestie\n" +
			"You are a supportive, enthusiastic best friend who ALSO happens to be a great developer. " +
			"Use a casual, warm tone. Sprinkle in emojis naturally (don't overdo it). " +
			"Celebrate wins and improvements. When pointing out bugs or issues, be kind and constructive — " +
			"phrase problems as 'hey, just noticed...' or 'might wanna check...' rather than blunt criticism. " +
			"Use Gen-Z friendly language where it fits naturally. Keep it real but keep it positive.\n\n"

	case model.PersonalitySigma:
		return "## 🪨 Your Reviewer Personality: The Sigma\n" +
			"You are the strong-silent type. Ultra terse. No greetings, no fluff, no small talk. " +
			"Every word earns its place. Use bullet points only. No preamble. " +
			"If there's nothing to say, say nothing. Facts only. Output the minimum necessary information " +
			"to convey the review. Do not explain what you're about to do — just do it.\n\n"

	case model.PersonalityToxicTechLead:
		return "## ☠️ Your Reviewer Personality: The Toxic Tech Lead\n" +
			"You are a brutally critical, sarcastic, and impatient tech lead who has seen it all and " +
			"has zero tolerance for sloppiness. Use sharp, cutting language. Mock obvious mistakes. " +
			"Be condescending about simple errors. Act like this PR physically pained you to read. " +
			"HOWEVER — you must remain technically accurate. Your snark must be backed by real, " +
			"correct technical reasoning. Never sacrifice correctness for cruelty. " +
			"Think: 'Gordon Ramsay reviews code'. Every bug should make you audibly sigh.\n\n"

	default: // PersonalitySeniorDev (default)
		return "## 🧑‍💻 Your Reviewer Personality: The Senior Dev\n" +
			"You are an experienced, professional senior engineer conducting a thorough code review. " +
			"Be clear, precise, and constructive. Point out issues with technical accuracy and " +
			"provide specific, actionable fixes. Be respectful but direct. " +
			"Prioritise correctness, maintainability, and security above all else.\n\n"
	}
}

// ── Base Prompts ──────────────────────────────────────────────────────────────

// systemPromptSummary instructs the model to produce a PR summary and per-file analysis.
var systemPromptSummary = "Your task is TWO things:\n" +
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
	"You have been given the exact code diff for each changed file, plus PR metadata and a short repository summary.\n\n" +
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
	"**API Contract Breakage**\n" +
	"- Interface implementors that now violate the updated interface\n" +
	"- Return type or signature change visible in the diff that would break callers\n\n" +
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

// ── Reviewer ────────────────────────────────────────────────────────────

// Reviewer executes 3 passes of LLM calls to produce a full code review via an
// OpenAI-compatible chat-completions endpoint.
type Reviewer struct {
	client *openai.Client
	log    *zap.Logger
	cfg    *config.Config
}

// NewReviewer creates a Reviewer using the provided API key.
func NewReviewer(cfg *config.Config, log *zap.Logger) *Reviewer {
	clientCfg := openai.DefaultConfig(cfg.LLM.APIKey)
	clientCfg.BaseURL = cfg.LLM.BaseURL

	c := openai.NewClientWithConfig(clientCfg)
	return &Reviewer{client: c, log: log, cfg: cfg}
}

// Review runs all three passes and returns a ReviewResult.
// The personality parameter controls the tone injected into all system prompts.
func (r *Reviewer) Review(ctx context.Context, rc *ReviewContext, personality model.Personality, enableArchMapping bool) (*ReviewResult, error) {
	result := &ReviewResult{}
	tone := personalityTone(personality)

	// ── Pass A: PR Summary + File Changes ──────────
	r.log.Info("reviewer: pass A — summary + file changes", zap.String("personality", string(personality)))
	summaryAndFiles, err := r.passA(ctx, rc, tone)
	if err != nil {
		return nil, fmt.Errorf("pass A failed: %w", err)
	}

	result.Summary, result.FileChanges = splitSummaryAndFiles(summaryAndFiles)
	r.log.Info("reviewer: pass A complete")

	// ── Pass B: Architecture Flow Diagram ──────────────
	if enableArchMapping {
		r.log.Info("reviewer: pass B — architecture diagram")
		diagram, err := r.passB(ctx, rc, tone)
		if err != nil {
			r.log.Warn("reviewer: pass B failed, skipping diagram", zap.Error(err))
			result.FlowDiagram = ""
		} else {
			result.FlowDiagram = diagram
			r.log.Info("reviewer: pass B complete")
		}
	} else {
		r.log.Info("reviewer: pass B skipped (Pro feature)")
		result.FlowDiagram = ""
	}

	// ── Pass C: Bug Detection ──────────────────────
	r.log.Info("reviewer: pass C — deep bug detection")
	bugReport, err := r.passC(ctx, rc, tone)
	if err != nil {
		return nil, fmt.Errorf("pass C failed: %w", err)
	}
	result.BugReport = bugReport
	r.log.Info("reviewer: pass C complete")

	return result, nil
}

// ── Pass A ────────────────────────────────────────────────────────────────────

func (r *Reviewer) passA(ctx context.Context, rc *ReviewContext, tone string) (string, error) {
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

	return r.CallLLM(ctx, tone+systemPromptSummary, userMsg, r.cfg.LLM.BaseModel)
}

// ── Pass B ────────────────────────────────────────────────────────────────────

func (r *Reviewer) passB(ctx context.Context, rc *ReviewContext, tone string) (string, error) {
	userMsg := fmt.Sprintf(`%s

Here are the changed files:
%s

Generate a Mermaid flowchart showing the NEW execution flow introduced by this PR.
Focus on the most architecturally significant change. Output ONLY the mermaid code block.`,
		rc.PRMeta,
		truncateToChars(rc.ChangedFilesBlock, 15_000),
	)

	return r.CallLLM(ctx, tone+systemPromptDiagram, userMsg, r.cfg.LLM.BaseModel)
}

// ── Pass C ────────────────────────────────────────────────────────────────────

func (r *Reviewer) passC(ctx context.Context, rc *ReviewContext, tone string) (string, error) {
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

	// Bug detection is the highest-value pass — use the stronger BugModel when set.
	model := r.cfg.LLM.BugModel
	if model == "" {
		model = r.cfg.LLM.BaseModel
	}
	return r.CallLLM(ctx, tone+systemPromptBugDetection, userMsg, model)
}

// ── API call helpers ──────────────────────────────────────────────────────────

func (r *Reviewer) CallLLM(ctx context.Context, systemPrompt, userMsg, model string) (string, error) {
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
		return "", fmt.Errorf("llm API: %w", err)
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
