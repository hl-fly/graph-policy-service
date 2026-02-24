package config

type Config struct {
	Server Server
}

type Server struct {
	Address string
}
