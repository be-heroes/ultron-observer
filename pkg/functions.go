package pkg

import (
	"fmt"
	"os"
	"strconv"

	attendant "github.com/be-heroes/ultron-attendant/pkg"
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
		KubernetesConfigPath: os.Getenv(attendant.EnvKubernetesConfig),
		KubernetesMasterURL:  fmt.Sprintf("tcp://%s:%s", os.Getenv(attendant.EnvKubernetesServiceHost), os.Getenv(attendant.EnvKubernetesServicePort)),
	}, nil
}

func InitializeKubernetesServiceFromConfig(config *Config) (kubernetesService services.IKubernetesService, err error) {
	kubernetesService, err = services.NewKubernetesService(config.KubernetesMasterURL, config.KubernetesConfigPath)
	if err != nil {
		return nil, err
	}

	return kubernetesService, nil
}
