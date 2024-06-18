package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/google/go-github/v62/github"
)

const promptFmtString = `The following text was parsed from a GitHub issue and its comments.
Extract the following information from the issue and comments:

- Issue: A list with the following items: title, link, the author name, the submission date and time.
- Summary: A summary of the issue in precisely one short sentence of no more than 50 words.
- Details: A longer summary of the issue. If code has been provided, list the pieces of code that cause the issue in the summary.
- Comments: A table with a summary of each comment in chronological order with the columns: date/time, time since the issue was submitted, author name, and a summary of the comment.
- Question: A table with a summary of each question that was posed to @%s by mention in the issue. Generate a suggested response to each question in the voice of a senior tech lead. The columns are: question and suggested response.

Don't waste words.
Don't include anything other than the requested sections.
Use short, clear, complete sentences.
Use active voice.
Maximize detail, meaning focus on the content.
Quote code snippets if they are relevant.
Answer in markdown with section headers separating each of the parts above.
Render author names a markdown links to the user's github profile page (i.e. https://github.com/username).

`

var (
	owner       string
	repo        string
	issueNumber int
	modelID     string
	outputDir   string
	verbose     bool
)

func init() {
	flag.StringVar(&owner, "owner", "", "GitHub owner name")
	flag.StringVar(&repo, "repo", "", "GitHub repository name")
	flag.IntVar(&issueNumber, "issue", 0, "GitHub issue number")
	flag.StringVar(&modelID, "model-id", "anthropic.claude-3-5-sonnet-20240620-v1:0", "AWS Bedrock model ID")
	flag.StringVar(&outputDir, "output-dir", "", "Output directory to write markdown to.")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
}

type Issue struct {
	Timestamp   time.Time
	URL         string
	AuthorName  string
	Username    string
	Title       string
	Description string
	Labels      []string
	Comments    []Comment
}

type Comment struct {
	Timestamp  time.Time
	AuthorName string
	Username   string
	Comment    string
}

func main() {
	flag.Parse()

	if owner == "" || repo == "" || issueNumber == 0 {
		flag.Usage()
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Build AWS API client.
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	bedrock := bedrockruntime.NewFromConfig(cfg)

	// Fetch GH issue and build the prompt.
	p, err := buildPrompt(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		fmt.Println("-------- Prompt -----------")
		fmt.Println(p)
		fmt.Println("-------- End Prompt -----------")
	}

	// Invoke Claude
	claudeResp, err := InvokeClaude(ctx, bedrock, p)
	if err != nil {
		log.Fatal(err)
	}

	if outputDir == "" {
		fmt.Println(claudeResp.Content[0].Text)
		return
	}

	markdownPath := filepath.Join(outputDir, owner, repo, strconv.Itoa(issueNumber)+".md")
	if err = os.MkdirAll(filepath.Dir(markdownPath), 0o700); err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(markdownPath, []byte(claudeResp.Content[0].Text), 0o600); err != nil {
		log.Fatal(err)
	}
}

func buildPrompt(ctx context.Context) (string, error) {
	// Build GitHub API client.
	client := github.NewClient(nil)
	client = client.WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	issue, err := issue(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed fetching github issue: %w", err)
	}

	ghUser, err := currentUser(client)
	if err != nil {
		return "", fmt.Errorf("failed fetching github user: %w", err)
	}

	prompt := new(strings.Builder)
	fmt.Fprintf(prompt, promptFmtString, ghUser)

	prompt.WriteString("<issue>\n")
	enc := json.NewEncoder(prompt)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(issue); err != nil {
		return "", fmt.Errorf("failed encoding github issue: %w", err)
	}
	prompt.WriteString("</issue>")
	prompt.WriteString("\n")

	return prompt.String(), nil
}

func currentUser(client *github.Client) (string, error) {
	user, _, err := client.Users.Get(context.TODO(), "")
	if err != nil {
		return "", err
	}

	return user.GetLogin(), nil
}

func issue(ctx context.Context, client *github.Client) (*Issue, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, issueNumber)
	if err != nil {
		return nil, err
	}

	var labels []string
	for _, l := range issue.Labels {
		labels = append(labels, l.GetName())
	}

	var authorName string
	if u, err := user(ctx, client, issue.User.GetLogin()); err == nil {
		authorName = u.GetName()
	}

	comments, err := comments(ctx, client)
	if err != nil {
		return nil, err
	}

	return &Issue{
		Timestamp:   issue.CreatedAt.UTC(),
		URL:         issue.GetHTMLURL(),
		AuthorName:  authorName,
		Username:    issue.User.GetLogin(),
		Title:       issue.GetTitle(),
		Description: issue.GetBody(),
		Labels:      labels,
		Comments:    comments,
	}, nil
}

func comments(ctx context.Context, client *github.Client) ([]Comment, error) {
	comments, _, err := client.Issues.ListComments(ctx, owner, repo, issueNumber, nil)
	if err != nil {
		return nil, err
	}

	out := make([]Comment, len(comments))
	for i, comment := range comments {
		var authorName string
		if u, err := user(ctx, client, comment.User.GetLogin()); err == nil {
			authorName = u.GetName()
		}

		out[i] = Comment{
			Timestamp:  comment.CreatedAt.UTC(),
			AuthorName: authorName,
			Username:   comment.User.GetLogin(),
			Comment:    comment.GetBody(),
		}
	}

	return out, nil
}

var usersCache = map[string]*github.User{}

func user(ctx context.Context, client *github.Client, login string) (*github.User, error) {
	if u := usersCache[login]; u != nil {
		return u, nil
	}

	u, _, err := client.Users.Get(ctx, login)
	if err != nil {
		return nil, err
	}
	usersCache[login] = u

	return u, nil
}

// -------------------------------
// AWS Bedrock helpers for Anthropic Claude
// -------------------------------

// Claude3Request represents the request payload for Claude 3 model.
// https://docs.anthropic.com/en/api/messages
type Claude3Request struct {
	Messages      []Message `json:"messages"`
	MaxTokens     int       `json:"max_tokens"`
	Temperature   float64   `json:"temperature,omitempty"`
	TopP          float64   `json:"top_p,omitempty"`
	TopK          int       `json:"top_k,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	System        string    `json:"system,omitempty"`
	Version       string    `json:"anthropic_version"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// InvokeModelResponse represents the response from the Claude 3 model
type InvokeModelResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func InvokeClaude(ctx context.Context, client *bedrockruntime.Client, prompt string) (*InvokeModelResponse, error) {
	body, err := json.Marshal(Claude3Request{
		Version:   "bedrock-2023-05-31",
		MaxTokens: 2048,
		System:    "You are an experienced software developer familiar with GitHub issues.",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error serializing claude request: %w", err)
	}

	output, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, err
	}

	var response InvokeModelResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
