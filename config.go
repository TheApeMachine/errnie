package errnie

type Config struct {
	Level string `mapstructure:"level"`
	File  struct {
		Active bool   `mapstructure:"active"`
		Path   string `mapstructure:"path"`
	} `mapstructure:"file"`
	Elasticsearch struct {
		Active   bool   `mapstructure:"active"`
		URL      string `mapstructure:"url"`
		Index    string `mapstructure:"index"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	} `mapstructure:"elasticsearch"`
}
