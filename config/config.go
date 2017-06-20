// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period       time.Duration `config:"period"`
	GithubToken  *string       `config:"github_token"`
	Repositories []Repository  `config:"repositories"`
}

type Repository struct {
	Account string `config:"account"`
	Name    string `config:"name"`
}

var DefaultConfig = Config{
	Period:       1 * time.Second,
	GithubToken:  nil,
	Repositories: []Repository{},
}
