package config

type ConfigInterface interface {
	Validate() error

	PostProcess()
}
