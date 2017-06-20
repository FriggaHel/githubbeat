package beater

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/FriggaHel/githubbeat/config"
	"github.com/google/go-github/github"

	"golang.org/x/oauth2"
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
	ctx := context.Background()

	if bt.config.GithubToken != nil {
		logp.Info("Using Github Token")
	}

	var tc *http.Client = nil
	if bt.config.GithubToken != nil {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *bt.config.GithubToken},
		)
		tc = oauth2.NewClient(ctx, ts)
	}
	client := github.NewClient(tc)

	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}

		for _, t := range bt.config.Repositories {
			rep, resp, err := client.Repositories.Get(ctx, t.Account, t.Name)
			if err != nil {
				logp.Warn(fmt.Sprintf("Unable to fetch data for %s/%s", t.Account, t.Name))
				continue
			}
			logp.Info(fmt.Sprintf("Remaining API calls: %d/%d", resp.Rate.Remaining, resp.Rate.Limit))
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
						"open_issues_count": *rep.OpenIssuesCount,
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
