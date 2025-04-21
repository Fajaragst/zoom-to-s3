package server

import (
	"context"

	"github.com/fajaragst/zoom-to-s3/internal/config"
	"github.com/fajaragst/zoom-to-s3/internal/delivery/http/handlers"
	"github.com/fajaragst/zoom-to-s3/internal/delivery/http/middleware"
	"github.com/fajaragst/zoom-to-s3/internal/service"
	"github.com/gin-gonic/gin"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Server struct {
	config *config.Config
	router *gin.Engine
}

func NewServer(cfg *config.Config) *Server {

	router := gin.Default()
	return &Server{
		config: cfg,
		router: router,
	}
}

func (s *Server) Run() error {

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	s3Client := s3.NewFromConfig(cfg)

	deps := service.Deps{
		S3: s3Client,
	}

	services := service.NewService(deps)
	handlers := handlers.NewHandlers(services)
	s.SetupRoutes(handlers)
	return s.router.Run(":" + s.config.Server.Port)
}

func (s *Server) SetupRoutes(h *handlers.Handlers) {
	// Create webhook validator middleware
	zoomValidator := middleware.NewZoomWebhookValidator(s.config.Zoom)

	v1 := s.router.Group("/api/v1")

	webhookGroup := v1.Group("/")
	webhookGroup.Use(zoomValidator)
	webhookGroup.POST("/", h.Record.UploadS3)
}
