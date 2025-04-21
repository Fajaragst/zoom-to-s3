package config

import (
	"os"
)

type ServerConfig struct {
	Port string
}

type AWSConfig struct {
	AccessKey   string
	SecretKey   string
	S3Bucket    string
	S3KeyPrefix string
}

type ZoomConfig struct {
	WebhookSecretToken string
}

type Config struct {
	Server ServerConfig
	AWS    AWSConfig
	Zoom   ZoomConfig
}

func NewConfig() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &Config{
		Server: ServerConfig{
			Port: port,
		},
		AWS: AWSConfig{
			AccessKey:   os.Getenv("AWS_ACCESS_KEY"),
			SecretKey:   os.Getenv("AWS_SECRET_KEY"),
			S3Bucket:    os.Getenv("AWS_S3_BUCKET"),
			S3KeyPrefix: os.Getenv("AWS_S3_KEY_PREFIX"),
		},
		Zoom: ZoomConfig{
			WebhookSecretToken: os.Getenv("ZOOM_WEBHOOK_SECRET_TOKEN"),
		},
	}, nil
}
