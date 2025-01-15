package internal

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	SrcDir string `yaml:"srcDir" env-default:"./monitor" env-required:"true"`
	//DestDir    string `yaml:"destDir" env-default:"./upload" env-required:"true"`
	//NumFolders int    `yaml:"numFolders" env-default:"2" env-required:"true"`
	Folders []string `yaml:"folders"`
}

func MustLoad() *Config {
	configPath := "./config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatal("can't read config: ", err)
	}
	return &cfg
}
