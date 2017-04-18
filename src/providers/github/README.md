# github

The github provider provides file storage. However, unlike other FS providers, github only supports pushing files in git repos. However, a push to github is atomic, and either works or doesn't work.

*This means that to make any FS changes, you must `git push` your changes.* You'll find `git log`, `add`, `commit`, `revert`, and `push` useful for this.

## Options

The `gh-username` and `gh-token` flag hold your github username and API token, respectively. You can generate an API token [here](https://github.com/settings/tokens). Be sure to give it full repo access.

The `gh-repo-name` flag is optional. If not included, you'll make or push to the repo by the name of the folder that the `stout create` command is run in.

## Config

The providers section of an example config file using github could look like the following:

```yaml
[...]
        providers:
                github:
                        gh-username: user
                        gh-token: testing
                        gh-repo-name: testrepo
```
