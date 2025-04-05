package app

type Config struct {
	AppHost                 string `env:"APP_HOST"`
	AppPort                 string `env:"APP_PORT"`
	EnableInactivityTimeout bool   `env:"ENABLE_INACTIVITY_TIMEOUT"`
	InactivityTimeout       int    `env:"INACTIVITY_TIMEOUT"`
	FlyRegion               string `env:"FLY_REGION"`
}
