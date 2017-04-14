package providers

type ConfigHolder map[string]EnvHolder

type EnvHolder struct {
	Env           string
	GlobalFlags   *GlobalFlags           `yaml:"global"`
	CreateFlags   *CreateFlags           `yaml:"create"`
	DeployFlags   *DeployFlags           `yaml:"deploy"`
	RollbackFlags *RollbackFlags         `yaml:"rollback"`
	ProviderFlags map[string]interface{} `yaml:"providers"`
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

type CreateFlags struct{}

type DeployFlags struct {
	Files string
	Root  string
	Dest  string
}

type RollbackFlags struct {
	Dest    string
	Version string `yaml:"-"`
}
