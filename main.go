package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/oauth2"
)

func main() {
	githubToken := getEnv("INPUT_GITHUB_TOKEN", "INPUT_GITHUB-TOKEN", "GH_TOKEN", "GITHUB_TOKEN")
	githubAPIBaseURL := getEnv("INPUT_GITHUB_API_BASE_URL", "INPUT_GITHUB-API-BASE-URL", "GH_API_BASE_URL")
	openaiAPIKey := getEnv("INPUT_OPENAI_API_KEY", "INPUT_OPENAI-API-KEY", "OPENAI_API_KEY")
	openaiModel := getEnv("INPUT_OPENAI_MODEL", "INPUT_OPENAI-MODEL", "OPENAI_MODEL")
	openaiBaseURL := getEnv("INPUT_OPENAI_BASE_URL", "INPUT_OPENAI-BASE-URL", "OPENAI_BASE_URL")
	maxTokensStr := getEnv("INPUT_MAX_TOKENS", "INPUT_MAX-TOKENS", "MAX_TOKENS")
	maxContextTokensStr := getEnv("INPUT_MAX_CONTEXT_TOKENS", "INPUT_MAX-CONTEXT-TOKENS", "MAX_CONTEXT_TOKENS")
	repoFullName := os.Getenv("GITHUB_REPOSITORY")
	eventPath := os.Getenv("GITHUB_EVENT_PATH")

	maxTokens := 2000
	if maxTokensStr != "" {
		if val, err := strconv.Atoi(maxTokensStr); err == nil {
			maxTokens = val
		}
	}

	maxContextTokens := 32000
	if maxContextTokensStr != "" {
		if val, err := strconv.Atoi(maxContextTokensStr); err == nil {
			maxContextTokens = val
		}
	}

	if openaiModel == "" {
		openaiModel = openai.GPT3Dot5Turbo
	}

	var missingVars []string
	if githubToken == "" {
		missingVars = append(missingVars, "github-token")
	}
	if openaiAPIKey == "" {
		missingVars = append(missingVars, "openai-api-key")
	}
	if repoFullName == "" {
		missingVars = append(missingVars, "GITHUB_REPOSITORY")
	}
	if eventPath == "" {
		missingVars = append(missingVars, "GITHUB_EVENT_PATH")
	}

	if len(missingVars) > 0 {
		fmt.Printf("::error::Missing required environment variables: %s\n", strings.Join(missingVars, ", "))
		os.Exit(1)
	}

	repoParts := strings.Split(repoFullName, "/")
	if len(repoParts) != 2 {
		fmt.Printf("::error::Invalid GITHUB_REPOSITORY format: %s\n", repoFullName)
		os.Exit(1)
	}
	owner, repo := repoParts[0], repoParts[1]

	prNumber, err := getPrNumber(eventPath)
	if err != nil {
		fmt.Printf("::error::%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("::debug::PR Number: %d\n", prNumber)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	var ghClient *github.Client
	if githubAPIBaseURL != "" && githubAPIBaseURL != "https://api.github.com" {
		ghClient, _ = github.NewClient(tc).WithEnterpriseURLs(githubAPIBaseURL, githubAPIBaseURL)
	} else {
		ghClient = github.NewClient(tc)
	}

	// OpenAI Client
	config := openai.DefaultConfig(openaiAPIKey)
	if openaiBaseURL != "" {
		config.BaseURL = openaiBaseURL
	}
	aiClient := openai.NewClientWithConfig(config)

	// Check for label
	fmt.Printf("::debug::Fetching PR details...\n")
	pr, _, err := ghClient.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		fmt.Printf("::error::Error fetching PR: %v\n", err)
		os.Exit(1)
	}

	hasLabel := false
	for _, l := range pr.Labels {
		if l.GetName() == "ai-describe" {
			hasLabel = true
			break
		}
	}

	if !hasLabel {
		fmt.Println("::info::Label 'ai-describe' not found. Skipping.")
		return
	}

	// Fetch changes
	fmt.Printf("::debug::Fetching PR files...\n")
	files, _, err := ghClient.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
	if err != nil {
		fmt.Printf("::error::Error fetching PR files: %v\n", err)
		os.Exit(1)
	}

	var changes strings.Builder
	// Heuristic: 1 token ~= 4 characters. We reserve some tokens for the prompt and response.
	maxChars := (maxContextTokens - maxTokens - 1000) * 4
	if maxChars < 0 {
		maxChars = 4000 // Fallback to a safe minimum
	}

	for _, file := range files {
		patch := file.GetPatch()
		fileDiff := fmt.Sprintf("File: %s\nChanges:\n%s\n", file.GetFilename(), patch)
		if changes.Len()+len(fileDiff) > maxChars {
			changes.WriteString(fmt.Sprintf("File: %s\nChanges:\n[Truncated due to context length limit]\n", file.GetFilename()))
			break
		}
		changes.WriteString(fileDiff)
	}

	fmt.Printf("::debug::Diff size: %d characters\n", changes.Len())

	// Fetch suggestions
	fmt.Printf("::info::Requesting AI suggestions from %s (this may take a while for large PRs)...\n", openaiModel)
	fmt.Printf("::debug::Requesting AI suggestions (model: %s, max_tokens: %d)...\n", openaiModel, maxTokens)
	suggestions, err := fetchAiSuggestions(ctx, aiClient, changes.String(), openaiModel, maxTokens)
	if err != nil {
		fmt.Printf("::error::Error fetching AI suggestions: %v\n", err)
		os.Exit(1)
	}

	// Update PR Body
	newBody := fmt.Sprintf("%s\n\n## PR Auto Describe\n%s", pr.GetBody(), suggestions)
	_, _, err = ghClient.PullRequests.Edit(ctx, owner, repo, prNumber, &github.PullRequest{
		Body: github.String(newBody),
	})
	if err != nil {
		fmt.Printf("::error::Error updating PR body: %v\n", err)
		// Fallback to comment if body update fails
		postComment(ctx, ghClient, owner, repo, prNumber, suggestions)
	} else {
		fmt.Printf("::info::Successfully updated PR #%d description.\n", prNumber)
	}
}

func getPrNumber(eventPath string) (int, error) {
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return 0, fmt.Errorf("error reading event file: %w", err)
	}

	var event struct {
		PullRequest struct {
			Number int `json:"number"`
		} `json:"pull_request"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return 0, fmt.Errorf("error unmarshaling event data: %w", err)
	}

	if event.PullRequest.Number == 0 {
		return 0, fmt.Errorf("pull request number not found in event data")
	}

	return event.PullRequest.Number, nil
}

func fetchAiSuggestions(ctx context.Context, client *openai.Client, prChanges string, model string, maxTokens int) (string, error) {
	prompt := generatePrompt(prChanges)

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.7,
			MaxTokens:   maxTokens,
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func generatePrompt(prChanges string) string {
	return fmt.Sprintf(`You are an expert software engineer writing a high-quality pull request summary.

Your goal is to produce a response that is BOTH:
- Easy to quickly understand (for fast reviewers)
- Fully detailed where it matters (for deep technical readers)

IMPORTANT:
- You have a LIMITED output budget.
- Prioritize clarity and coverage of ALL important changes.
- If necessary, compress less important details, but NEVER omit critical functionality changes.

---

### INPUT: Changes Breakdown
%s

---

### OUTPUT FORMAT (STRICT)

## 🚀 Summary
(2–4 sentences, MAX ~400 characters)

---

## 🔑 Key Changes
(Concise, grouped, MAX ~30%% of total output)

---

## 📋 Detailed Breakdown
- Include ALL important technical changes
- Prefer compact phrasing over long explanations
- Merge closely related low-level changes when needed
- Use dense but readable descriptions

---

## 🧠 Notes (Optional)
Only if critical.

---

## ⚠️ Breaking Changes (If any)
Only if these changes break existing functionality or APIs.

---

### STYLE GUIDELINES
- Be concise but information-dense
- Avoid redundancy between sections
- Prioritize important changes over minor ones
- Always highlight any BREAKING CHANGES in a dedicated sub-section if detected.
`, prChanges)
}

func postComment(ctx context.Context, client *github.Client, owner, repo string, prNumber int, suggestions string) {
	commentBody := fmt.Sprintf("**PR Auto Describe:**\n\n%s", suggestions)
	_, _, err := client.Issues.CreateComment(ctx, owner, repo, prNumber, &github.IssueComment{
		Body: github.String(commentBody),
	})
	if err != nil {
		fmt.Printf("::error::Error posting comment: %v\n", err)
	} else {
		fmt.Printf("::info::Successfully posted comment to PR #%d.\n", prNumber)
	}
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}
