package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/fajaragst/zoom-to-s3/internal/service"
	"github.com/gin-gonic/gin"
)

type RecordHandlers struct {
	recordService service.RecordService
}

func NewRecordHandlres(recordService service.RecordService) *RecordHandlers {
	return &RecordHandlers{
		recordService: recordService,
	}
}

func (h *RecordHandlers) UploadS3(c *gin.Context) {
	var input service.Record

	// Print the headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s3Bucket := os.Getenv("AWS_S3_BUCKET")
	s3KeyPrefix := os.Getenv("AWS_S3_KEY_PREFIX")

	go h.recordService.UploadS3(c, input, s3Bucket, s3KeyPrefix)

	c.JSON(http.StatusOK, gin.H{"message": "file processing started"})
}
