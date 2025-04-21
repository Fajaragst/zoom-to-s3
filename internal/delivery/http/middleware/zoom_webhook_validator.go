package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fajaragst/zoom-to-s3/internal/config"
	"github.com/gin-gonic/gin"
)

type ZoomWebhookConfig struct {
	SecretToken string
	Tolerance   time.Duration
}

func NewZoomWebhookValidator(config config.ZoomConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Processing Zoom webhook request")

		// 1. Get the Zoom verification headers
		signature := c.GetHeader("X-Zm-Signature")
		timestamp := c.GetHeader("X-Zm-Request-Timestamp")

		if signature == "" || timestamp == "" {
			log.Println("Missing Zoom verification headers")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing Zoom verification headers",
			})
			return
		}

		log.Printf("Received headers - Signature: %s, Timestamp: %s", signature, timestamp)

		// Read and restore the request body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read request body",
			})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Validate the signature
		message := fmt.Sprintf("v0:%s:%s", timestamp, string(bodyBytes))
		log.Printf("Message to hash: %s", message)

		// Create HMAC SHA-256 hash
		mac := hmac.New(sha256.New, []byte(config.WebhookSecretToken))
		mac.Write([]byte(message))
		hashForVerify := hex.EncodeToString(mac.Sum(nil))

		// Format the expected signature
		expectedSignature := fmt.Sprintf("v0=%s", hashForVerify)
		log.Printf("Expected signature: %s", expectedSignature)

		// Compare signatures
		if signature != expectedSignature {
			log.Printf("Invalid signature. Expected: %s, Got: %s", expectedSignature, signature)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid webhook signature",
			})
			return
		}

		log.Println("Zoom webhook signature validated successfully")
		c.Next()
	}
}
