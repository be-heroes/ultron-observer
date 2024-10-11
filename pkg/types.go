package pkg

type Config struct {
	RedisServerAddress   string
	RedisServerPassword  string
	RedisServerDatabase  int
	KubernetesConfigPath string
	KubernetesMasterURL  string
}
