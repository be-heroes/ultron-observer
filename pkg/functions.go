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
		RedisServerDatabase:  redisDatabase,
		KubernetesConfigPath: os.Getenv(ultron.EnvKubernetesConfig),
	}, nil
}

func InitializeKubernetesClient(config *Config) (kubernetesClient services.IKubernetesService, err error) {
	kubernetesMasterUrl := fmt.Sprintf("tcp://%s:%s", os.Getenv(ultron.EnvKubernetesServiceHost), os.Getenv(ultron.EnvKubernetesServicePort))

	kubernetesClient, err = services.NewKubernetesClient(kubernetesMasterUrl, config.KubernetesConfigPath)
	if err != nil {
		return nil, err
	}

	return kubernetesClient, nil
}
