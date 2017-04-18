package github

import (
	"errors"
	"os"
	"strings"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/github/fs"
)

func (cl *client) CreateFS(g providers.GlobalFlags, c providers.CreateFlags) (string, error) {
	reponame := cl.Reponame
	if reponame == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return "", err
		}

		folders := strings.Split(pwd, "/")
		reponame = folders[len(folders)-1]
	}

	err := fs.Create(ghclient, ctx, g.Domain, cl.Token, reponame, cl.Username)
	if err != nil {
		return "", err
	}

	fsDomain := cl.Username + ".github.io"
	return fsDomain, nil
}

func (cl *client) DeployFS(g providers.GlobalFlags, c providers.DeployFlags) error {
	return errors.New("Using git to push to the gh-pages branch, to deploy to Github pages.")
}

func (cl *client) RollbackFS(g providers.GlobalFlags, c providers.RollbackFlags) error {
	return errors.New("Use git to revert to and push a previous commit, to revert on Github pages.")
}
