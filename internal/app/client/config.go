package client

import (
	"flag"
	"os"

	domainservices "github.com/DyadyaRodya/GophKeeper/internal/domain/services"
)

// Config stores all config data for App
type Config struct {
	GrpcAddress string
	Debug       bool
	Loglevel    string

	UsernameConfig           *domainservices.UsernameConfig
	PasswordComplexityConfig *domainservices.PasswordComplexityConfig
}

// InitConfigFromCMD reads CMD line and env arguments to Config
func InitConfigFromCMD(
	defaultGrpcAddress string,
	defaultDebug bool,
) *Config {
	grpcAddress := flag.String("g", defaultGrpcAddress, "grpc address to bind")
	debug := flag.Bool("d", defaultDebug, "enable debug output")
	flag.BoolVar(debug, "debug", defaultDebug, "enable debug output")
	flag.Parse()

	if envGrpcAddress := os.Getenv("GRPC_ADDRESS"); envGrpcAddress != "" {
		grpcAddress = &envGrpcAddress
	}

	if envDebug := os.Getenv("DEBUG"); envDebug != "" {
		*debug = true
	}

	logLevel := "info"
	if *debug {
		logLevel = "debug"
	}
	return &Config{
		GrpcAddress: *grpcAddress,
		Debug:       *debug,
		Loglevel:    logLevel,

		UsernameConfig: &domainservices.UsernameConfig{
			MinLen: 8,
			MaxLen: 50,
			AllowedChars: []rune{
				'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
				'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
				'0', '1', '2', '3', '4', '5', '7', '8', '9', '_',
			},
		},
		PasswordComplexityConfig: &domainservices.PasswordComplexityConfig{
			Length:          8,
			NumberOfDigits:  1,
			NumberOfUpper:   1,
			NumberOfLower:   1,
			NumberOfSpecial: 1,
		},
	}
}
