package utils

import (
	"flag"
	"strings"

	"github.com/urfave/cli"
)

type titleFlag struct {
	Title  string
	Hidden bool
}

func (t titleFlag) Apply(*flag.FlagSet) {}
func (t titleFlag) String() string      { return "\n" + t.Title }
func (t titleFlag) GetName() string     { return "" }

func TitleFlag(title string) cli.Flag {
	return titleFlag{Title: title}
}

func FormattedUsageText() string {
	text := (`
stout [global options] <command> [command options], or
stout help <command>, to learn more about a subcommand

Example Usage:

To create a site which will be hosted at my.awesome.website:
  stout create --fs=amazon --cdn=amazon --dns=amazon --domain=my.awesome.website --key=AWS_KEY --secret=AWS_SECRET

To deploy the current folder to the root of the my.awesome.website site:
  stout deploy --fs=amazon --domain=my.awesome.website --key=AWS_KEY --secret=AWS_SECRET

To rollback to a specific deploy:
  stout rollback --fs=amazon --domain=my.awesome.website --key=AWS_KEY --secret=AWS_SECRET c4a22bf94de1
 `)

	textArray := strings.Split(text, "\n")
	formattedText := strings.Join(textArray[1:], "\n   ")

	return formattedText
}
