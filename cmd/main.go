package main

import (
	"log"

	"github.com/fleetcard/config"
	"github.com/fleetcard/controllers"
	"github.com/fleetcard/db"
	"github.com/fleetcard/services"
)

func main() {
	// โหลด config จาก .env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// เชื่อมต่อฐานข้อมูล
	db.DB = db.Connect(cfg)
	
	// ดาวน์โหลด .gpg จาก SFTP และเก็บใน EncryptedReport
	controllers.DownloadAllGPGFromSFTP()

	// ดึงไฟล์ล่าสุดที่โหลดมา
	var report db.EncryptedReport
	if err := db.DB.Order("created_at desc").First(&report).Error; err != nil {
		log.Fatalf("No encrypted report found: %v", err)
	}

	// ถอดรหัส + แปลงไฟล์ csv → เป็น []Transaction
	txs, err := services.DecryptAndExtractCSV(report, cfg.DateFormat)
	if err != nil {
		log.Fatalf("Failed to decrypt and extract CSV: %v", err)
	}

	// บันทึกข้อมูล Transaction ลงฐานข้อมูล
	if err := services.SaveTransactions(txs); err != nil {
		log.Fatalf("Failed to save transactions: %v", err)
	}

	log.Printf("Successfully imported %d transactions from: %s", len(txs), report.FileName)
}
