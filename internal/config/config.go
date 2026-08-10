package config

type Config struct {
	StoreLocation string
}

func NewDummyConfig() Config {
	return Config{
		StoreLocation: "/tmp/rido-dummy-store",
	}
}
