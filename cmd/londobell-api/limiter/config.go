package limiter

type LimitConfRules struct {
	Rules map[string]*LimitOpt `mapstructure:"rules"`
}

type LimitOpt struct {
	Interval int64 `mapstructure:"interval"`
	Capacity int64 `mapstructure:"capacity"`
	Quantum  int64 `mapstructure:"quantum"`
}
