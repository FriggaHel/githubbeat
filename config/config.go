// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period       time.Duration `config:"period"`
	Step         time.Duration `config:"step"`
	GithubToken  *string       `config:"github_token"`
	FetchPast    bool          `config:"fetch_past"`
	Repositories []Repository  `config:"repositories"`
}

type Repository struct {
	Account string `config:"account"`
	Name    string `config:"name"`
}

var DefaultConfig = Config{
	Period:       1 * time.Second,
	Step:         1 * time.Hour,
	GithubToken:  nil,
	FetchPast:    false,
	Repositories: []Repository{},
}
