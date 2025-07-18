package awstagprocessor

type Config struct {
    CacheFile string `mapstructure:"cache_file"`
    TTL       int    `mapstructure:"ttl"`
}
