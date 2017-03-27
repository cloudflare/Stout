package types

type GlobalFlags struct {
	Config string
	Env    string
	Domain string
	DNS    string
	FS     string
	CDN    string
}

type CreateFlags struct {
	CreateSSL bool
	NoSSL     bool
}

type DeployFlags struct {
	Files string
	Root  string
	Dest  string
}

type RollbackFlags struct {
	Dest    string
	Version string
}
