package providers

type ConfigHolder map[string]EnvHolder

type EnvHolder struct {
	Env           string
	GlobalFlags   *GlobalFlags
	CreateFlags   *CreateFlags
	DeployFlags   *DeployFlags
	RollbackFlags *RollbackFlags
	ProviderFlags map[string]interface{}
}

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
	Version string `yaml:"-"`
}
