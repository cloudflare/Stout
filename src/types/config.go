package types

type ConfigHolder map[string]EnvHolder

type EnvHolder struct {
	Env           string
	GlobalFlags   *GlobalFlags           `yaml:"global"`
	CreateFlags   *CreateFlags           `yaml:"create"`
	DeployFlags   *DeployFlags           `yaml:"deploy"`
	RollbackFlags *RollbackFlags         `yaml:"rollback"`
	ProviderFlags map[string]interface{} `yaml:"providers"`
}
