package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/FriggaHel/githubbeat/config"
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
				"github": common.MapStr{
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
