package fs

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/eagerio/Stout/src/providers/github/git"
	"github.com/google/go-github/github"
)

func Create(ghclient *github.Client, ctx context.Context, domain string, token string, reponame string, username string) error {
	if !git.Exists() {
		err := git.Exec("init")
		if err != nil {
			return err
		}
	}

	ghclient.Repositories.Create(ctx, "", &github.Repository{
		Name:    github.String(reponame),
		Private: github.Bool(false),
	})

	// github requires the subdomain to be in a CNAME file for custom domains
	err := ioutil.WriteFile("./CNAME", []byte(domain), 0644)
	if err != nil {
		return err
	}

	err = git.ChainExecLog([][]string{
		[]string{"checkout", "-b", "gh-pages"},

		[]string{"add", "./CNAME"},
		[]string{"commit", "-m", `"Stout initial commit."`},

		[]string{"remote", "add", "origin", fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", username, token, username, reponame)},
		[]string{"push", "--set-upstream", "origin", "gh-pages"},
	})
	if err != nil {
		return err
	}

	return nil
}
