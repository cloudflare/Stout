package types

type GlobalFlags struct {
	Debug  bool   `yaml:"-"`
	Config string `yaml:"-"`
	Env    string `yaml:"-"`
	Domain string
	DNS    string
	FS     string
	CDN    string
}

type CreateFlags struct {
	DomainValidationHelp bool
}

type DeployFlags struct {
	Files string
	Root  string
	Dest  string
}

type RollbackFlags struct {
	Dest    string
	Version string `yaml:"-"`
}
