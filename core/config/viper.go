package config

import "github.com/spf13/viper"

type ViperConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Must bool   `json:"must"`
}

// 调用AutomaticEnv函数，开启环境变量读取
func ViperInitEnv() {
	viper.AutomaticEnv()
}

func LoadViperConfig(config ViperConfig) error {
	viper.SetConfigName(config.Name)
	viper.SetConfigType(config.Type)
	viper.AddConfigPath(config.Path)
	err := viper.MergeInConfig()
	if err != nil {
		if config.Must {
			panic(err)
		}
		return err
	}
	return nil
}
