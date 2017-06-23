package beater

import (
	"context"
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Repository struct {
	Account      string
	RepoName     string
	GithubClient *github.Client
	GithubToken  *string
	Context      context.Context
	HttpClient   *http.Client
	LogQuotas    bool
	Repo         *github.Repository
	Stargazers   []*github.Stargazer
	Forks        []*github.Repository
	Issues       []*github.Issue
}

func NewRepository(bt *Githubbeat, account string, repo string) *Repository {
	return &Repository{
		Account:      account,
		RepoName:     repo,
		GithubClient: nil,
		GithubToken:  bt.config.GithubToken,
		Context:      context.Background(),
		HttpClient:   nil,
		LogQuotas:    true,
		Repo:         nil,
		Stargazers:   []*github.Stargazer{},
		Forks:        []*github.Repository{},
		Issues:       []*github.Issue{},
	}
}

func (r *Repository) GetGithubClient() *github.Client {
	if r.GithubClient == nil {
		if r.GithubToken != nil {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: *r.GithubToken},
			)
			r.HttpClient = oauth2.NewClient(r.Context, ts)
		}
		r.GithubClient = github.NewClient(r.HttpClient)
	}
	return r.GithubClient
}

func (r *Repository) GetRepo() (*github.Repository, error) {
	if r.Repo == nil {
		client := r.GetGithubClient()
		logp.Info(fmt.Sprintf("Fetching Repository Info for %s/%s", r.Account, r.RepoName))
		repo, resp, err := client.Repositories.Get(r.Context, r.Account, r.RepoName)
		if err != nil {
			return r.Repo, err
		}
		r.Repo = repo
		if r.LogQuotas {
			logp.Info(fmt.Sprintf("Remaining: %d/%d", resp.Rate.Remaining, resp.Rate.Limit))
		}
	}
	return r.Repo, nil
}

func (r *Repository) GetForks() ([]*github.Repository, error) {
	if len(r.Forks) == 0 {
		client := r.GetGithubClient()

		opt := &github.RepositoryListForksOptions{
			Sort: "oldest",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
		logp.Info(fmt.Sprintf("Fetching forks for %s/%s", r.Account, r.RepoName))
		for {
			forks, resp, err := client.Repositories.ListForks(r.Context, r.Account, r.RepoName, opt)
			if err != nil {
				return r.Forks, err
			}
			if r.LogQuotas {
				logp.Info(fmt.Sprintf("Remaining: %d/%d", resp.Rate.Remaining, resp.Rate.Limit))
			}
			r.Forks = append(r.Forks, forks...)
			if resp.NextPage == 0 {
				break
			}
			opt.ListOptions.Page = resp.NextPage
		}
	}
	return r.Forks, nil
}

func (r *Repository) GetIssues(state string) ([]*github.Issue, error) {
	if len(r.Issues) == 0 {
		client := r.GetGithubClient()

		opt := &github.IssueListByRepoOptions{
			State:     state,
			Sort:      "created",
			Direction: "asc",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
		logp.Info(fmt.Sprintf("Fetching Issues/PR for %s/%s", r.Account, r.RepoName))
		for {
			issues, resp, err := client.Issues.ListByRepo(r.Context, r.Account, r.RepoName, opt)
			if err != nil {
				return r.Issues, err
			}
			if r.LogQuotas {
				logp.Info(fmt.Sprintf("Remaining: %d/%d", resp.Rate.Remaining, resp.Rate.Limit))
			}
			r.Issues = append(r.Issues, issues...)

			if resp.NextPage == 0 {
				break
			}
			opt.ListOptions.Page = resp.NextPage
		}
	}
	return r.Issues, nil
}

func (r *Repository) GetStargazers() ([]*github.Stargazer, error) {
	if len(r.Stargazers) == 0 {
		client := r.GetGithubClient()

		opt := &github.ListOptions{
			PerPage: 100,
		}
		logp.Info(fmt.Sprintf("Fetching Stargazers for %s/%s", r.Account, r.RepoName))
		for {
			gazers, resp, err := client.Activity.ListStargazers(r.Context, r.Account, r.RepoName, opt)
			if err != nil {
				return r.Stargazers, err
			}
			if r.LogQuotas {
				logp.Info(fmt.Sprintf("Remaining: %d/%d", resp.Rate.Remaining, resp.Rate.Limit))
			}
			r.Stargazers = append(r.Stargazers, gazers...)

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	return r.Stargazers, nil
}
