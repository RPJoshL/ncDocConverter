package models

import (
	"flag"
	"os"

	yaml "gopkg.in/yaml.v3"
	"rpjosh.de/ncDocConverter/pkg/logger"
)

type WebConfig struct {
	Server  Server  `yaml:"server"`
	Logging Logging `yaml:"logging"`
}

type Server struct {
	Address     string `yaml:"address"`
	Certificate string `yaml:"certificate"`
	OneShot     bool   `yaml:"oneShot"`
}

type Logging struct {
	PrintLogLevel string `yaml:"printLogLevel"`
	WriteLogLevel string `yaml:"writeLogLevel"`
	LogFilePath   string `yaml:"logFilePath"`
}

// Parses the configuration file (.yaml file) to an WebConfiguration
func ParseWebConfig(webConfig *WebConfig, file string) (*WebConfig, error) {
	if file == "" {
		return webConfig, nil
	}

	dat, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(dat, &webConfig); err != nil {
		return nil, err
	}

	return webConfig, nil
}

func getDefaultConfig() *WebConfig {
	return &WebConfig{
		Server: Server{
			Address: ":4000",
		},
		Logging: Logging{
			PrintLogLevel: "info",
			WriteLogLevel: "warning",
		},
	}
}

// Applies the cli and the configuration options from the config files
func SetConfig() (*WebConfig, error) {
	configPath := "./config.yaml"
	// the path of the configuration file is needed first to determine the "default" values
	for i, arg := range os.Args {
		if arg == "-config" || arg == "--config" && len(os.Args) > i {
			configPath = os.Args[i+1]
			break
		}
	}
	webConfig := getDefaultConfig()
	webConfig, err := ParseWebConfig(webConfig, configPath)
	if err != nil {
		logger.Error("Unable to parse the configuration file '%s': %s", configPath, err)
		webConfig = getDefaultConfig()
		err = nil
	}

	_ = flag.String("config", "./config.yaml", "Path to the configuration file (see configs/config.yaml) for an example")
	address := flag.String("address", webConfig.Server.Address, "Address and port on which the api and the web server should listen to")
	printLogLevel := flag.String("printLogLevel", webConfig.Logging.PrintLogLevel, "Minimum log level to log (debug, info, warning, error, fatal)")
	oneShot := flag.Bool("oneShot", webConfig.Server.OneShot, "All jobs are executed immediately and the program exists afterwards")

	flag.Parse()
	webConfig.Server.Address = *address
	webConfig.Logging.PrintLogLevel = *printLogLevel
	webConfig.Server.OneShot = *oneShot

	defaultLogger := logger.Logger{
		PrintLevel:  logger.GetLevelByName(webConfig.Logging.PrintLogLevel),
		LogLevel:    logger.GetLevelByName(webConfig.Logging.WriteLogLevel),
		LogFilePath: webConfig.Logging.LogFilePath,
		PrintSource: true,
	}
	logger.SetGlobalLogger(&defaultLogger)

	return webConfig, err
}
