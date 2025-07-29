package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const githubAPI = "https://api.github.com"

type Event struct {
	Type      string    `json:"type"`
	Repo      Repo      `json:"repo"`
	Payload   Payload   `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type Repo struct {
	Name string `json:"name"`
}

type Payload struct {
	Action      string       `json:"action"`
	Comment     *Comment     `json:"comment,omitempty"`
	Issue       *Issue       `json:"issue,omitempty"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
	Commits     []Commit     `json:"commits,omitempty"`
	Review      *Review      `json:"review,omitempty"`
}

type Comment struct {
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

type Issue struct {
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
}

type PullRequest struct {
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
}

type Commit struct {
	Message string `json:"message"`
	SHA     string `json:"sha"`
}

type Review struct {
	State string `json:"state"`
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}

	username, err := getAuthenticatedUsername(token)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Fetching events for:", username)

	eventsByDate := make(map[string][]Event)
	page := 1

	for {
		events, err := fetchEvents(username, token, page)
		if err != nil {
			log.Fatal("Failed to fetch events:", err)
		}
		if len(events) == 0 {
			break
		}
		for _, event := range events {
			date := event.CreatedAt.Local().Format("2006-01-02")
			eventsByDate[date] = append(eventsByDate[date], event)
		}
		page++
	}

	writeEvents(eventsByDate)
}

func getAuthenticatedUsername(token string) (string, error) {
	login, err := getUser(token)
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return login, nil
}

func getUser(token string) (string, error) {
	req, _ := http.NewRequest("GET", githubAPI+"/user", nil)
	req.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "", err
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}
	return user.Login, nil
}

func fetchEvents(username, token string, page int) ([]Event, error) {
	url := fmt.Sprintf("%s/users/%s/events?page=%d&per_page=100", githubAPI, username, page)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch events (non-200 response): got HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var events []Event
	if err = json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}
	return events, nil
}

func writeEvents(eventsByDate map[string][]Event) {
	for date, events := range eventsByDate {
		filename := filepath.Join(os.Getenv("OUTPUT_DIR"), fmt.Sprintf("%s-github_activity.md", date))
		if _, err := os.Stat(filename); err == nil {
			log.Println("File already exists, skipping:", filename)
			continue
		}

		f, err := os.Create(filename)
		if err != nil {
			log.Println("Failed to write file:", err)
			continue
		}
		// Close the file at the end of processing this date's events
		func() {
			defer f.Close()

			fmt.Fprintf(f, "# GitHub Activity for %s\n\n", date)

			// Group by repo
			repoMap := make(map[string][]Event)
			for _, ev := range events {
				repoMap[ev.Repo.Name] = append(repoMap[ev.Repo.Name], ev)
			}

			repos := make([]string, 0, len(repoMap))
			for r := range repoMap {
				repos = append(repos, r)
			}
			sort.Strings(repos)

			for _, repo := range repos {
				fmt.Fprintf(f, "## %s\n\n", repo)

				// Group by type within repo
				sections := map[string][]string{
					"Pull Requests": {},
					"Issues":        {},
					"Commits":       {},
					"Other":         {},
				}

				for _, ev := range repoMap[repo] {
					timestamp := ev.CreatedAt.Local().Format("15:04:05")
					switch ev.Type {
					case "PullRequestEvent":
						if pr := ev.Payload.PullRequest; pr != nil {
							line := fmt.Sprintf(
								"### [#%d](%s): %s\n- **PR %s** at %s\n",
								pr.Number, pr.HTMLURL, html.EscapeString(pr.Title), cases.Title(language.English).String(ev.Payload.Action), timestamp,
							)
							sections["Pull Requests"] = append(sections["Pull Requests"], line)
						}
					case "PullRequestReviewEvent":
						if pr := ev.Payload.PullRequest; pr != nil {
							review := ev.Payload.Review
							disposition := "Review"
							if review != nil {
								disposition = cases.Title(language.English).String(review.State)
							}
							line := fmt.Sprintf(
								"### [#%d](%s): %s\n- **Review:** %s at %s\n",
								pr.Number, pr.HTMLURL, html.EscapeString(pr.Title), disposition, timestamp,
							)
							sections["Pull Requests"] = append(sections["Pull Requests"], line)
						}
					case "PullRequestReviewCommentEvent":
						comment := ev.Payload.Comment
						if comment != nil {
							body := html.EscapeString(truncate(comment.Body, 50))
							line := fmt.Sprintf("- **Inline Comment**: \"%s\" [view](%s) at %s\n", body, comment.HTMLURL, timestamp)
							sections["Pull Requests"] = append(sections["Pull Requests"], line)
						}
					case "IssueCommentEvent":
						issue := ev.Payload.Issue
						comment := ev.Payload.Comment
						if issue != nil && comment != nil {
							body := html.EscapeString(truncate(comment.Body, 50))
							line := fmt.Sprintf(
								"### [#%d](%s): %s\n- **Comment**: \"%s\" [view](%s) at %s\n",
								issue.Number, issue.HTMLURL, html.EscapeString(issue.Title), body, comment.HTMLURL, timestamp,
							)
							sections["Issues"] = append(sections["Issues"], line)
						}
					case "IssuesEvent":
						issue := ev.Payload.Issue
						if issue != nil {
							line := fmt.Sprintf(
								"### [#%d](%s): %s\n- **%s Issue** at %s\n",
								issue.Number, issue.HTMLURL, html.EscapeString(issue.Title), cases.Title(language.English).String(ev.Payload.Action), timestamp,
							)
							sections["Issues"] = append(sections["Issues"], line)
						}
					case "PushEvent":
						for _, c := range ev.Payload.Commits {
							url := fmt.Sprintf("https://github.com/%s/commit/%s", repo, c.SHA)
							message := "````\n" + c.Message + "\n````"
							line := fmt.Sprintf("- **Commit** [%s](%s) at %s\n%s\n", c.SHA[:7], url, timestamp, message)
							sections["Commits"] = append(sections["Commits"], line)
						}
					default:
						line := fmt.Sprintf("- **%s** at %s\n", ev.Type, timestamp)
						sections["Other"] = append(sections["Other"], line)
					}
				}

				// Write sections
				for _, section := range []string{"Pull Requests", "Issues", "Commits", "Other"} {
					if len(sections[section]) > 0 {
						fmt.Fprintf(f, "### %s\n\n", section)
						for _, line := range sections[section] {
							fmt.Fprint(f, line+"\n")
						}
					}
				}
			}

			fmt.Println("✅ Wrote:", filename)
		}()
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
