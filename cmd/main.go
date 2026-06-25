package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"edc-pinpad/config"
	"edc-pinpad/delivery"
	"edc-pinpad/infrastructure"
	"edc-pinpad/middleware"
	"edc-pinpad/pkg/logger"
	"edc-pinpad/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// STRUKTUR MANIFEST UNTUK GATEWAY
type ConfigManifest struct {
	Key         string      `json:"key"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
}

type RegisterRequest struct {
	Name           string           `json:"name"`
	Address        string           `json:"address"`
	ConfigManifest []ConfigManifest `json:"config_manifest"`
}

func main() {
	_ = godotenv.Load()

	l := logger.SetupLogger("edc_pinpad_gateway")
	slog.SetDefault(l)

	l.Info("EDC Pinpad Gateway starting...")

	config.GlobalConfig.LoadInitialEnv()

	var listener net.Listener
	var err error
	var db *gorm.DB
	dbReady := make(chan bool)

	if config.GlobalConfig.Standalone {
		l.Info("🚀 Mode: STANDALONE (Menggunakan .env)")

		listener, err = net.Listen("tcp", ":"+strconv.Itoa(config.GlobalConfig.ListenPort))
		if err != nil {
			l.Error("Failed to get port", slog.String("error", err.Error()))
			os.Exit(1)
		}

		// Init DB langsung karena standalone punya infra sendiri di .env
		db = infrastructure.InitDB(l, config.GlobalConfig)

		// TAMBAHAN: Jalankan Watcher agar param bisnis di DB lokal bisa Hot-Reload
		go startLocalConfigWatcher(db, 10*time.Second, l)
	} else {
		// =========================================================
		// 1. DAPATKAN PORT MASTER (Dari .env warisan)
		// =========================================================
		masterAddr := os.Getenv("MASTER_URL")
		if masterAddr == "" {
			// Fallback ke mode Monolith (Warisan dari Master)
			masterPort := os.Getenv("APP_PORT")
			if masterPort == "" {
				masterPort = "8080"
			}
			masterAddr = "http://127.0.0.1:" + masterPort
		}

		// =========================================================
		// 2. DAPATKAN PORT DINAMIS UNTUK MODUL INI (Port 0 Magic)
		// =========================================================
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			l.Error("Failed to get dynamic port from OS", slog.String("error", err.Error()))
			os.Exit(1)
		}

		// Ekstrak port riil yang baru saja diberikan oleh OS
		actualPort := listener.Addr().(*net.TCPAddr).Port
		myAddr := fmt.Sprintf("http://localhost:%d", actualPort)
		config.GlobalConfig.ListenPort = actualPort

		// =========================================================
		// 3. HANDSHAKE KE MASTER DENGAN ALAMAT DINAMIS
		// =========================================================
		go registerToGateway(l, "pinpad", masterAddr, myAddr)
		<-dbReady
		db = infrastructure.GetDB()
	}

	ws := infrastructure.NewWSManager(l, db)
	uc := usecase.NewPinpadUsecase(db, ws, l)
	h := delivery.NewPinpadHandler(uc, ws, l)

	r := setupRouter(l, h)
	// =========================================================
	// 4. JALANKAN SERVER GIN MENGGUNAKAN LISTENER
	// =========================================================
	l.Info("EDC Pinpad Gateway listening", slog.Int("port", config.GlobalConfig.ListenPort))

	// Kita JANGAN pakai r.Run(":9001") lagi, tapi pakai r.RunListener
	// agar Gin menggunakan jalur jaringan yang sudah kita booking di atas.
	if err := r.RunListener(listener); err != nil {
		l.Error("EDC Pinpad Gateway failed to start", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func setupRouter(l *slog.Logger, h *delivery.PinpadHandler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Route dinamis: Handler baru dibuat saat request masuk agar selalu dapat instance DB terbaru
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "not found"})
	})

	// =========================================================
	// 1. PASANG MIDDLEWARE SECARA GLOBAL (Berlaku untuk semua rute)
	// =========================================================
	r.Use(func(c *gin.Context) {
		db := infrastructure.GetDB()
		if db == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "System initializing, database not ready"})
			return
		}
		c.Next()
	})

	if config.GlobalConfig.Standalone {
		a := r.Group("/api/pinpad")

		a.POST("/create-pin", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePinCreate(c)
		})

		a.POST("/change-pin", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePinChange(c)
		})

		a.POST("/verify-pin", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePinVerify(c)
		})

		a.POST("/penarikan-tunai", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePenarikanTunai(c)
		})

		a.POST("/transfer", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandleTransfer(c)
		})

		// Endpoint WebSocket untuk EDC fisik (sudah disesuaikan 1 TID 10 EDC ya)
		wsGroup := a.Group("/ws")
		wsGroup.Use(middleware.HMACAuth()) // Middleware untuk otentikasi HMAC
		{
			wsGroup.GET("/edc", func(c *gin.Context) {
				h.HandleWS(c)
			})
		}
	} else {
		// Endpoint untuk Master Gateway
		r.POST("/create-pin", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePinCreate(c)
		})

		r.POST("/change-pin", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePinChange(c)
		})

		r.POST("/verify-pin", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePinVerify(c)
		})

		r.POST("/penarikan-tunai", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandlePenarikanTunai(c)
		})

		r.POST("/transfer", middleware.GatewayAuth(), func(c *gin.Context) {
			h.HandleTransfer(c)
		})

		// Endpoint WebSocket untuk EDC fisik (sudah disesuaikan 1 TID 10 EDC ya)
		wsGroup := r.Group("/ws")
		wsGroup.Use(middleware.HMACAuth()) // Middleware untuk otentikasi HMAC
		{
			wsGroup.GET("/edc", func(c *gin.Context) {
				h.HandleWS(c)
			})
		}
	}

	return r
}

// Watcher untuk mode Standalone agar param bisnis bisa Hot-Reload dari DB Lokal
func startLocalConfigWatcher(db *gorm.DB, interval time.Duration, l *slog.Logger) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		config.GlobalConfig.SyncFromLocalDB(db) // Fungsi ini sudah kita bahas sebelumnya
	}
}

// registerToGateway: Mengirim info ke Master Gateway agar dikenali
func registerToGateway(l *slog.Logger, name, masterAddr, myAddr string) {
	isRegistered := false
	url := masterAddr + "/internal/register"

	// BACA DARI SINGLE SOURCE OF TRUTH DI PACKAGE CONFIG
	var manifest []ConfigManifest
	for _, dp := range config.DefaultParameters {
		manifest = append(manifest, ConfigManifest{
			Key:         dp.Key,
			Type:        dp.Type,
			Default:     dp.DefaultVal, // Tetap di-passing sebagai tipe aslinya (int/string)
			Description: dp.Description,
		})
	}

	payload := RegisterRequest{
		Name:           name,
		Address:        myAddr,
		ConfigManifest: manifest,
	}

	for {
		jsonPayload, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		req.Header.Set("Content-Type", "application/json")

		// AMBIL WARISAN DARI MASTER
		internalSecret := os.Getenv("INTERNAL_GATEWAY_SECRET")
		req.Header.Set("X-Internal-Secret", internalSecret)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			if !isRegistered {
				l.Info("✅ Successfully linked to Master Gateway",
					slog.String("module", name),
					slog.String("my_addr", myAddr),
					slog.String("master", masterAddr),
				)
				isRegistered = true
			}
			var result struct {
				Configs map[string]string `json:"configs"`
			}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			// UPDATE CONFIG DI RAM (Hot-Reload)
			config.GlobalConfig.Mu.Lock()
			if val, ok := result.Configs["DEVICE_SECRET"]; ok {
				config.GlobalConfig.DeviceSecret = val
			}
			if val, ok := result.Configs["CLIENT_SECRET"]; ok {
				config.GlobalConfig.ClientSecret = val
			}
			if val, ok := result.Configs["HOST_URL"]; ok {
				config.GlobalConfig.HostURL = val
			}
			if val, ok := result.Configs["DB_HOST"]; ok {
				config.GlobalConfig.DbHost = val
			}
			if val, ok := result.Configs["DB_PORT"]; ok {
				config.GlobalConfig.DbPort = val
			}
			if val, ok := result.Configs["DB_USER"]; ok {
				config.GlobalConfig.DbUser = val
			}
			if val, ok := result.Configs["DB_PASS"]; ok {
				config.GlobalConfig.DbPass = val
			}
			if val, ok := result.Configs["DB_NAME"]; ok {
				config.GlobalConfig.DbName = val
			}
			config.GlobalConfig.Mu.Unlock()

			if infrastructure.GetDB() == nil {
				infrastructure.InitDB(l, config.GlobalConfig)
			}

			time.Sleep(10 * time.Second) // Heartbeat setiap menit
		} else {
			isRegistered = false
			l.Warn("⏳ Master Gateway not reachable. Retrying in 5 seconds...", slog.String("master", masterAddr))
			time.Sleep(5 * time.Second)
		}
	}
}
