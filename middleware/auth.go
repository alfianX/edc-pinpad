package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"edc-pinpad/config"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func HMACAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Akses GlobalConfig dari package config
		config.GlobalConfig.Mu.RLock()
		secret := config.GlobalConfig.DeviceSecret
		config.GlobalConfig.Mu.RUnlock()

		tid := c.GetHeader("X-TID")
		ts := c.GetHeader("X-Timestamp")
		sig := c.GetHeader("X-Signature")

		if tid == "" || ts == "" || sig == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Security headers missing"})
			return
		}

		// Check timestamp (5 min tolerance)
		tInt, _ := strconv.ParseInt(ts, 10, 64)
		if time.Now().Unix()-tInt > 300 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Timestamp expired"})
			return
		}

		// Verify HMAC
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(tid + ts))
		expected := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(sig), []byte(expected)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}
		c.Next()
	}
}

// GatewayAuth: Validasi HMAC-SHA512 untuk request dari Master Gateway
func GatewayAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Akses GlobalConfig dari package config
		config.GlobalConfig.Mu.RLock()
		clientSecret := config.GlobalConfig.ClientSecret
		config.GlobalConfig.Mu.RUnlock()

		partnerID := c.GetHeader("x-partner-id")
		timestamp := c.GetHeader("x-timestamp")
		signatureReceived := c.GetHeader("x-signature")

		if partnerID == "" || timestamp == "" || signatureReceived == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing auth headers"})
			return
		}

		// Spek: stringToSign = X-PARTNER-ID+X-TIMESTAMP
		stringToSign := partnerID + timestamp

		mac := hmac.New(sha512.New, []byte(clientSecret))
		mac.Write([]byte(stringToSign))
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signatureReceived), []byte(expectedSignature)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}
		c.Next()
	}
}
