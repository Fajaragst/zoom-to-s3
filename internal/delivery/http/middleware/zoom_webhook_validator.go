package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fajaragst/zoom-to-s3/internal/config"
	"github.com/gin-gonic/gin"
)

func NewZoomWebhookValidator(config config.ZoomConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Processing Zoom webhook request")

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

		// Check if this is a URL validation request
		var requestBody map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &requestBody); err == nil {
			if event, ok := requestBody["event"].(string); ok && event == "endpoint.url_validation" {
				handleURLValidation(c, requestBody, config.WebhookSecretToken)
				return
			}
		}

		// For regular webhook requests, validate the signature
		signature := c.GetHeader("X-Zm-Signature")
		timestamp := c.GetHeader("X-Zm-Request-Timestamp")

		if signature == "" || timestamp == "" {
			log.Println("Missing Zoom verification headers")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing Zoom verification headers",
			})
			return
		}

		// Validate the signature
		message := fmt.Sprintf("v0:%s:%s", timestamp, string(bodyBytes))

		// Create HMAC SHA-256 hash
		mac := hmac.New(sha256.New, []byte(config.WebhookSecretToken))
		mac.Write([]byte(message))
		hashForVerify := hex.EncodeToString(mac.Sum(nil))

		// Format the expected signature
		expectedSignature := fmt.Sprintf("v0=%s", hashForVerify)

		// Compare signatures
		if signature != expectedSignature {
			log.Printf("Invalid signature")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid webhook signature",
			})
			return
		}

		c.Next()
	}
}

// handleURLValidation processes Zoom's URL validation request
func handleURLValidation(c *gin.Context, requestBody map[string]interface{}, secretToken string) {
	log.Println("Processing URL validation request")

	// Extract plainToken from the payload
	if payload, ok := requestBody["payload"].(map[string]interface{}); ok {
		if plainToken, ok := payload["plainToken"].(string); ok {
			// Create HMAC SHA-256 hash of the plainToken
			mac := hmac.New(sha256.New, []byte(secretToken))
			mac.Write([]byte(plainToken))
			encryptedToken := hex.EncodeToString(mac.Sum(nil))

			// Return the validation response
			c.JSON(http.StatusOK, gin.H{
				"plainToken":     plainToken,
				"encryptedToken": encryptedToken,
			})
			c.Abort()
			return
		}
	}

	// If we couldn't extract the plainToken
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"error": "Invalid URL validation request",
	})
}
