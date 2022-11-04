package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"golang.org/x/sys/execabs"
)

const checkInterval = 60 * time.Second

var (
	owner  string
	repo   string
	commit string
)

func init() {
	flag.StringVar(&owner, "owner", "", "repo owner (example: elastic)")
	flag.StringVar(&repo, "repo", "", "repo name (example: package-storage)")
	flag.StringVar(&commit, "commit", "", "commit hash (example: 57eeb0c)")
}

func main() {
	flag.Parse()

	if owner == "" || repo == "" || commit == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	var httpClient *http.Client

	if token := os.Getenv("GITHUB_AUTH_TOKEN"); token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(httpClient)

	for state := range watch(ctx, client, owner, repo, commit) {
		switch state {
		case "success":
			notify("Github PR Success", "Your PR can be merged.")
		case "failure":
			notify("Github PR Failure", "Your PR failed CI.")
		case "pending":
		default:
			panic("invalid state")
		}
	}
}

func notify(title, message string) error {
	// terminal-notifier -title "Github PR Ready" -message "Your PR can be merged." -group github -open "https://github.com/elastic/integrations/pull/3353" -sound Funk
	cmd := execabs.Command("terminal-notifier",
		"-title", title,
		"-message", message,
		"-group", "pr-watch",
		"-sound", "Funk")

	_, err := cmd.CombinedOutput()
	return err
}

func watch(ctx context.Context, client *github.Client, owner, repo, commit string) <-chan string {
	stateChan := make(chan string, 1)

	go func() {
		defer close(stateChan)

		combinedState, err := checkStatus(ctx, client, owner, repo, commit)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(combinedState)
		stateChan <- combinedState

		tick := time.NewTicker(checkInterval)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				combinedState, err := checkStatus(ctx, client, owner, repo, commit)
				if err != nil {
					log.Println(err)
					continue
				}

				fmt.Println(combinedState)
				stateChan <- combinedState
			}
		}
	}()

	return stateChan
}

func checkStatus(ctx context.Context, client *github.Client, owner, repo, commit string) (state string, err error) {
	status, resp, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, commit, &github.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get combined status for commit %v: %w", commit, err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		fmt.Println("Limit", resp.Limit)
		fmt.Println("Remaining", resp.Remaining)
		fmt.Println("Reset", resp.Reset)

		delay := time.Until(resp.Reset.Time)
		log.Printf("Waiting for %v for rate-limit reset.", delay)
		time.Sleep(delay)

		return "", fmt.Errorf("rate-limited")
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed with status code %v", resp.StatusCode)
	}

	if status == nil || status.State == nil {
		return "", fmt.Errorf("missing combined state in response")
	}

	return *status.State, nil
}
