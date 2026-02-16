package selfupdate

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"gitee.com/naibahq/go-gitee/gitee"
)

const atomGitBaseURL = "https://api.atomgit.com/api"

type AtomGitUpdater struct {
	api        *gitee.APIClient
	apiCtx     context.Context
	validator  Validator
	filters    []*regexp.Regexp
	binaryName string
}

func NewAtomGitUpdater(config Config) (*AtomGitUpdater, error) {
	token := config.APIToken
	if token == "" {
		token = os.Getenv("ATOMGIT_TOKEN")
	}
	ctx := context.Background()

	filtersRe := make([]*regexp.Regexp, 0, len(config.Filters))
	for _, filter := range config.Filters {
		re, err := regexp.Compile(filter)
		if err != nil {
			return nil, fmt.Errorf("Could not compile regular expression %q for filtering releases: %v", filter, err)
		}
		filtersRe = append(filtersRe, re)
	}

	conf := gitee.NewConfiguration()
	conf.BasePath = atomGitBaseURL
	conf.HTTPClient = newHTTPClient(ctx, token)

	client := gitee.NewAPIClient(conf)
	return &AtomGitUpdater{api: client, apiCtx: ctx, validator: config.Validator, filters: filtersRe, binaryName: config.BinaryName}, nil
}

func DefaultAtomGitUpdater() *AtomGitUpdater {
	token := os.Getenv("ATOMGIT_TOKEN")
	ctx := context.Background()
	conf := gitee.NewConfiguration()
	conf.BasePath = atomGitBaseURL
	conf.HTTPClient = newHTTPClient(ctx, token)
	return &AtomGitUpdater{api: gitee.NewAPIClient(conf), apiCtx: ctx}
}
