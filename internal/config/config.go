// Package config has a configuration structure
package config

// Config contains configuration data
type Config struct {
	UsernamePostgres string `env:"POSTGRES_USER" envDefault:"postgres"`
	PasswordPostgres string `env:"POSTGRES_PASSWORD" envDefault:"testpassword"`
	HostPostgres     string `env:"POSTGRES_USER" envDefault:"localhost"`
	PortPostgres     string `env:"POSTGRES_USER" envDefault:"5432"`
	DBNamePostgres   string `env:"POSTGRES_DB" envDefault:"postgres"`

	ServerRedisCache string `env:"SERVER" envDefault:"server1"`
	HostRedisCache   string `env:"HOST" envDefault:"localhost"`
	PortRedisCache   string `env:"PORT" envDefault:"6379"`

	HostGrpcClient string `env:"HOST_GRPC" envDefault:"localhost"`
	PortGrpcClient string `env:"PORT_GRPC" envDefault:"10000"`

	HostGrpcServer string `env:"HOST_GRPC_SERVER" envDefault:"localhost"`
	PortGrpcServer string `env:"PORT_GRPC_SERVER" envDefault:"11000"`
}
