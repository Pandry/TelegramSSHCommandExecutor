package config

type config struct {
	Telegram     telegram
	Commands     map[string]features `toml:"features"`
	AllowedUsers map[string]allowedUsers
	KnownServers map[string]knownservers
	Settings     settings
}

type telegram struct {
	TelegramAPIToken string `toml:"TelegramAPIToken"`
}

type settings struct {
	DefaultUsername string
	DefaultPassword string
	Debug bool
}

type knownservers struct {
	IP       string
	Username string
	Password string
}

type features struct {
	Commands       []string
	ExpectedOutputs []string
}

type allowedUsers struct {
	ID       int
	UserName string
}
