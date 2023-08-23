package builtin

type Scheduler struct {
	Job  string `yaml:"job"`
	Cron string `yaml:"cron"`
	Task struct {
		URL    string `yaml:"url"`
		Method string `yaml:"method"`
		Body   string `yaml:"body"`
		Header string `yaml:"header"`
	} `yaml:"task"`
}
