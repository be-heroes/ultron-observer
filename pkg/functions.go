package pkg

import (
	"fmt"
	"os"
	"strconv"

	ultron "github.com/be-heroes/ultron/pkg"
	services "github.com/be-heroes/ultron/pkg/services"
)

func LoadConfig() (*Config, error) {
	redisDatabase, err := strconv.Atoi(os.Getenv(ultron.EnvRedisServerDatabase))
	if err != nil {
		redisDatabase = 0
	}

	return &Config{
		RedisServerAddress:   os.Getenv(ultron.EnvRedisServerAddress),
		RedisServerPassword:  os.Getenv(ultron.EnvRedisServerPassword),
		RedisServerDatabase:  redisDatabase,
		KubernetesConfigPath: os.Getenv(ultron.EnvKubernetesConfig),
		KubernetesMasterUrl:  fmt.Sprintf("tcp://%s:%s", os.Getenv(ultron.EnvKubernetesServiceHost), os.Getenv(ultron.EnvKubernetesServicePort)),
	}, nil
}

func InitializeKubernetesServiceFromConfig(config *Config) (kubernetesService services.IKubernetesService, err error) {
	kubernetesService, err = services.NewKubernetesService(config.KubernetesMasterUrl, config.KubernetesConfigPath)
	if err != nil {
		return nil, err
	}

	return kubernetesService, nil
}
