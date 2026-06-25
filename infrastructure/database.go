package infrastructure

import (
	"database/sql"
	"edc-pinpad/config"
	"edc-pinpad/model"
	"fmt"
	"log"
	"log/slog"

	_ "github.com/go-sql-driver/mysql" // Driver MySQL murni untuk raw query
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var dbInstance *gorm.DB

func InitDB(l *slog.Logger, cfg *config.AppConfig) *gorm.DB {
	cfg.Mu.RLock()
	user, pass, host, port, dbName := cfg.DbUser, cfg.DbPass, cfg.DbHost, cfg.DbPort, cfg.DbName
	cfg.Mu.RUnlock()

	// 1. KONEKSI AWAL (Tanpa Nama Database)
	// Kita connect ke MySQL server secara umum dulu
	dsnNoDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/", user, pass, host, port)
	rawDB, err := sql.Open("mysql", dsnNoDB)
	if err != nil {
		l.Error("Failed to connect to MySQL server", slog.String("error", err.Error()))
		log.Fatal("Gagal akses MySQL Server: ", err)
	}
	defer rawDB.Close()

	// 2. CREATE DATABASE JIKA BELUM ADA
	_, err = rawDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	if err != nil {
		l.Error("Failed to create database", slog.String("error", err.Error()))
		log.Fatal("Gagal membuat database: ", err)
	}

	// 3. KONEKSI GORM (Dengan Nama Database)
	// Sekarang database sudah pasti ada, kita bisa pakai GORM seperti biasa
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		l.Error("Failed to connect to MySQL database with GORM", slog.String("error", err.Error()))
		log.Fatal("GORM Connection Error: ", err)
	}

	// 4. AUTOMIGRATE
	// Membuat tabel sesuai struct model
	err = db.AutoMigrate(&model.Transaction{}, &model.WsConnection{})
	if err != nil {
		l.Error("Failed to auto-migrate database", slog.String("error", err.Error()))
		log.Fatal("AutoMigrate Error: ", err)
	}

	dbInstance = db
	l.Info("Database initialized successfully", slog.String("database", dbName))
	return db
}

func GetDB() *gorm.DB {
	return dbInstance
}
