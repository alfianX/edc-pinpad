package config

import (
	"edc-pinpad/model"
	"fmt"
	"os"
	"strconv"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 1. STRUKTUR CONFIG LOKAL (Internal State)
type AppConfig struct {
	Mu           sync.RWMutex
	ListenPort   int    `json:"LISTEN_PORT"`
	Standalone   bool   `json:"STANDALONE"`
	PartnerID    string `json:"PARTNER_ID"`
	ClientSecret string `json:"CLIENT_SECRET"`
	DeviceSecret string `json:"DEVICE_SECRET"`
	HostURL      string `json:"HOST_URL"`
	DbHost       string `json:"DB_HOST"`
	DbPort       string `json:"DB_PORT"`
	DbUser       string `json:"DB_USER"`
	DbPass       string `json:"DB_PASS"`
	DbName       string `json:"DB_NAME"`
	LicenseKey   string `json:"LICENSE_KEY"`
}

var GlobalConfig = &AppConfig{}

type DefaultParam struct {
	Key         string
	Type        string
	DefaultVal  interface{} // Pakai interface{} agar bisa nampung int maupun string
	Description string
}

var DefaultParameters = []DefaultParam{
	{"PARTNER_ID", "string", "MYBANK-PARTNER-01", "ID unik untuk mengidentifikasi mitra (Bank Aceh)"},
	{"DEVICE_SECRET", "string", "RahasiaEDC123", "Kunci HMAC EDC"},
	{"CLIENT_SECRET", "string", "SecretBank789!", "Kunci HMAC untuk request dari Master"},
	{"HOST_URL", "string", "http://127.0.0.1:8888", "URL Core Banking / Host tujuan"},
}

func (cfg *AppConfig) LoadInitialEnv() {
	cfg.Mu.Lock()
	defer cfg.Mu.Unlock()

	// Cek apakah mode standalone atau nempel master
	cfg.Standalone = os.Getenv("STANDALONE_MODE") == "true"

	if cfg.Standalone {
		// =========================================================
		// 1. LOAD PARAMETER BISNIS (Otomatis dari DefaultParameters)
		// =========================================================
		for _, dp := range DefaultParameters {
			// Coba ambil dari file .env
			envVal := os.Getenv(dp.Key)

			// Jika di .env kosong/tidak ditulis, gunakan nilai Default-nya
			if envVal == "" {
				envVal = fmt.Sprintf("%v", dp.DefaultVal)
			}

			// Masukkan ke dalam struct AppConfig
			switch dp.Key {
			case "HOST_URL":
				cfg.HostURL = envVal
			case "CLIENT_SECRET":
				cfg.ClientSecret = envVal
			case "DEVICE_SECRET":
				cfg.DeviceSecret = envVal
			case "PARTNER_ID":
				cfg.PartnerID = envVal
			}
		}

		// =========================================================
		// 2. LOAD PARAMETER SISTEM (Wajib dari .env)
		// =========================================================
		// Parameter ini tidak dimasukkan ke DefaultParameters karena
		// sifatnya rahasia (Database) atau spesifik untuk jaringan OS (Port)
		if port, err := strconv.Atoi(os.Getenv("LISTEN_PORT")); err == nil {
			cfg.ListenPort = port
		}

		cfg.DbHost = os.Getenv("DB_HOST")
		cfg.DbPort = os.Getenv("DB_PORT")
		cfg.DbUser = os.Getenv("DB_USER")
		cfg.DbPass = os.Getenv("DB_PASS")
		cfg.DbName = os.Getenv("DB_NAME")
		cfg.LicenseKey = os.Getenv("LICENSE_KEY")
	}
}

func (cfg *AppConfig) SyncFromLocalDB(db *gorm.DB) {
	cfg.Mu.Lock()
	defer cfg.Mu.Unlock()

	// 1. Tarik dari Single Source of Truth dan ubah ke format Database
	var defaultParams []model.SysParameter
	for _, dp := range DefaultParameters {
		// fmt.Sprintf("%v") akan otomatis mengubah angka 60 menjadi string "60" untuk DB
		valStr := fmt.Sprintf("%v", dp.DefaultVal)
		defaultParams = append(defaultParams, model.SysParameter{
			ParamKey:    dp.Key,
			ParamType:   dp.Type,
			ParamValue:  valStr,
			Description: dp.Description,
		})
	}

	// 2. AutoMigrate & Seed (Opsional)
	db.AutoMigrate(&model.SysParameter{})
	for _, p := range defaultParams {
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(&p)
	}

	// 3. Tarik semua dari DB ke RAM (GlobalConfig)
	var params []model.SysParameter
	db.Find(&params)

	for _, p := range params {
		switch p.ParamKey {
		case "DEVICE_SECRET":
			cfg.DeviceSecret = p.ParamValue
		case "CLIENT_SECRET":
			cfg.ClientSecret = p.ParamValue
		case "HOST_URL":
			cfg.HostURL = p.ParamValue
		case "PARTNER_ID":
			cfg.PartnerID = p.ParamValue
		}
	}
}
