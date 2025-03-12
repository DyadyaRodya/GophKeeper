package server

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	domainservices "github.com/DyadyaRodya/GophKeeper/internal/domain/services"
)

// Config stores all config data for App
type Config struct {
	GrpcAddress string
	LogLevel    string
	StorageDir  string
	DSN         string
	EnableHTTPS bool

	TokenTTL time.Duration

	SaltSize int

	UsernameConfig           *domainservices.UsernameConfig
	PasswordComplexityConfig *domainservices.PasswordComplexityConfig
}

type config struct {
	GrpcAddress string `json:"grpc_address"`
	LogLevel    string `json:"log_level"`
	StorageDir  string `json:"storage_dir"`
	DSN         string `json:"database_dsn"`
	EnableHTTPS bool   `json:"enable_https"`
}

// InitConfigFromCMD reads  config file, CMD line and env arguments to Config
func InitConfigFromCMD(
	defaultGrpcAddress,
	defaultLogLevel,
	defaultStorageDir string,
) *Config {
	configFile := flag.String("c", "", "path to config file")
	flag.StringVar(configFile, "config", "", "path to config file")

	grpcAddress := flag.String("g", "", "grpc address to bind")
	logLevel := flag.String("l", "", "log level")
	storageDir := flag.String("f", "", "file storage dir")
	dsn := flag.String("d", "", "database connection string")
	enableHTTPS := flag.Bool("s", false, "enable https")
	flag.Parse()

	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		configFile = &envConfigFile
	}

	if envGrpcAddress := os.Getenv("GRPC_ADDRESS"); envGrpcAddress != "" {
		grpcAddress = &envGrpcAddress
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		logLevel = &envLogLevel
	}
	if envStorageDir := os.Getenv("STORAGE_DIR"); envStorageDir != "" {
		storageDir = &envStorageDir
	}
	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		dsn = &envDSN
	}
	if envEnableHTTPS := os.Getenv("ENABLE_HTTPS"); envEnableHTTPS != "" {
		*enableHTTPS = true
	}

	conf := &config{}
	if *configFile != "" {
		f, err := os.Open(*configFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		err = json.NewDecoder(f).Decode(conf)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *grpcAddress == "" {
		if conf.GrpcAddress != "" {
			*grpcAddress = conf.GrpcAddress
		} else {
			*grpcAddress = defaultGrpcAddress
		}
	}

	if *logLevel == "" {
		if conf.LogLevel != "" {
			*logLevel = conf.LogLevel
		} else {
			*logLevel = defaultLogLevel
		}
	}
	if *storageDir == "" {
		if conf.StorageDir != "" {
			*storageDir = conf.StorageDir
		} else {
			*storageDir = defaultStorageDir
		}
	}
	if *dsn == "" {
		*dsn = conf.DSN
	}
	if !*enableHTTPS {
		*enableHTTPS = conf.EnableHTTPS
	}

	return &Config{
		GrpcAddress: *grpcAddress,
		LogLevel:    *logLevel,
		StorageDir:  *storageDir,
		DSN:         *dsn,
		EnableHTTPS: *enableHTTPS,

		// values for tests
		TokenTTL: 0 * time.Hour,
		SaltSize: 16,
		UsernameConfig: &domainservices.UsernameConfig{
			MinLen: 4,
			MaxLen: 50,
			AllowedChars: []rune{
				'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
				'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
				'0', '1', '2', '3', '4', '5', '7', '8', '9', '_',
			},
		},
		PasswordComplexityConfig: &domainservices.PasswordComplexityConfig{
			Length:          0,
			NumberOfDigits:  0,
			NumberOfUpper:   0,
			NumberOfLower:   0,
			NumberOfSpecial: 0,
		},
	}
}
