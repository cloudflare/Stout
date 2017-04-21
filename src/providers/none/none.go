package none

import "github.com/urfave/cli"

var Client client

type client struct{}

func (c *client) Name() string            { return "none" }
func (c *client) Flags() []cli.Flag       { return []cli.Flag{} }
func (c *client) ValidateSettings() error { return nil }
