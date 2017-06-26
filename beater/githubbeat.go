package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/FriggaHel/githubbeat/config"
	"github.com/google/go-github/github"
)

type Githubbeat struct {
	done   chan struct{}
	config config.Config
	client publisher.Client
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Githubbeat{
		done:   make(chan struct{}),
		config: config,
	}
	return bt, nil
}

func (bt *Githubbeat) Run(b *beat.Beat) error {
	logp.Info("githubbeat is running! Hit CTRL-C to stop it.")

	bt.client = b.Publisher.Connect()
	ticker := time.NewTicker(bt.config.Period)

	if bt.config.GithubToken != nil {
		logp.Info("Using Github Token")
	}

	if bt.config.FetchPast == true {
		bt.FetchPast()
	}

	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		for _, t := range bt.config.Repositories {
			r := NewRepository(bt, t.Account, t.Name)
			rep, err := r.GetRepo()
			if err != nil {
				logp.Warn(fmt.Sprintf("Unable to fetch data for %s/%s", t.Account, t.Name))
				continue
			}

			// Count PRs
			issues, err := r.GetIssues("open")
			prCount := 0
			for _, is := range issues {
				if is.PullRequestLinks != nil {
					prCount++
				}
			}

			event := common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"type":       b.Name,
				"repository": common.MapStr{
					"fullname":          fmt.Sprintf("%s/%s", t.Account, t.Name),
					"account":           t.Account,
					"name":              t.Name,
					"forks_count":       *rep.ForksCount,
					"network_count":     *rep.NetworkCount,
					"open_issues_count": *rep.OpenIssuesCount - prCount,
					"open_pr_count":     prCount,
					"stargazers_count":  *rep.StargazersCount,
					"subscribers_count": *rep.SubscribersCount,
					"watchers_count":    *rep.WatchersCount,
				},
			}
			bt.client.PublishEvent(event)
		}
	}
	return nil
}

func (bt *Githubbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}

func (bt *Githubbeat) FetchPast() {
	for _, t := range bt.config.Repositories {
		r := NewRepository(bt, t.Account, t.Name)

		// Fetch Forks + Issues + Stars
		forks, _ := r.GetForks()
		issues, _ := r.GetIssues("all")
		stargazers, _ := r.GetStargazers()

		// Beginning + Short rewind
		cur := bt.FindFirstDate(stargazers, issues, forks).Add(bt.config.Step * -1)

		// Indexes
		var stargazersIdx int = 0
		var forksIdx int = 0
		var issuesIdx int = 0

		now := time.Now()
		for {
			stargazersCount := bt.CountStargazers(cur, stargazers, &stargazersIdx)
			issuesCount, prCount := bt.CountIssues(cur, issues, &issuesIdx)
			forksCount := bt.CountForks(cur, forks, &forksIdx)

			event := common.MapStr{
				"@timestamp": common.Time(cur),
				"type":       "githubbeat",
				"repository": common.MapStr{
					"fullname":          fmt.Sprintf("%s/%s", t.Account, t.Name),
					"account":           t.Account,
					"name":              t.Name,
					"forks_count":       forksCount,
					"network_count":     0,
					"open_issues_count": issuesCount,
					"open_pr_count":     prCount,
					"stargazers_count":  stargazersCount,
					"subscribers_count": 0,
					"watchers_count":    0,
				},
			}
			bt.client.PublishEvent(event)
			if cur.After(now) {
				break
			}
			cur = cur.Add(bt.config.Step)
		}
	}
}

func (bt *Githubbeat) CountForks(t time.Time, forks []*github.Repository, index *int) int {
	for *index < len(forks) && forks[*index].CreatedAt.Time.Before(t) {
		(*index)++
	}
	return *index
}

func (bt *Githubbeat) CountIssues(t time.Time, issues []*github.Issue, index *int) (int, int) {
	issueCnt := 0
	prCnt := 0

	for *index < len(issues) && issues[*index].CreatedAt.Before(t) {
		(*index)++
	}

	// Keep It safe
	rIndex := *index
	if rIndex == len(issues) {
		rIndex--
	}
	for ; rIndex >= 0; rIndex-- {
		if issues[rIndex].ClosedAt == nil || issues[rIndex].ClosedAt.After(t) {
			if issues[rIndex].PullRequestLinks == nil {
				issueCnt++
			} else {
				prCnt++
			}
		}
	}
	return issueCnt, prCnt
}

func (bt *Githubbeat) CountStargazers(t time.Time, stargazers []*github.Stargazer, index *int) int {
	for *index < len(stargazers) && stargazers[*index].StarredAt.Time.Before(t) {
		(*index)++
	}
	return *index
}

func (bt *Githubbeat) FindFirstDate(stargazers []*github.Stargazer, issues []*github.Issue, forks []*github.Repository) time.Time {
	t := time.Now()

	// Assume this is sorted (api seems to)
	if len(stargazers) > 0 && stargazers[0].StarredAt.Time.Before(t) {
		t = stargazers[0].StarredAt.Time
	}

	// Assume they are sorted => https://developer.github.com/v3/issues/#list-issues-for-a-repository
	if len(issues) > 0 && issues[0].CreatedAt.Before(t) {
		t = *issues[0].CreatedAt
	}

	// Assume this is sorted => https://developer.github.com/v3/repos/forks/#list-forks
	if len(forks) > 0 && forks[0].CreatedAt.Before(t) {
		t = forks[0].CreatedAt.Time
	}
	return t
}
