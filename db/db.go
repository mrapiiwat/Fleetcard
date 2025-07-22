package db

import (
	"log"
	"time"

	"github.com/fleetcard/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type Transaction struct {
	AccountNo              string    `gorm:"column:account_no" json:"account_no"`
	AccountName            string    `gorm:"column:account_name" json:"account_name"`
	FleetCardNumber        string    `gorm:"column:fleet_card_number" json:"fleet_card_number"`
	LicensePlateNumber     string    `gorm:"column:license_plate_number" json:"license_plate_number"`
	CardholderName         string    `gorm:"column:cardholder_name" json:"cardholder_name"`
	CostCenter             string    `gorm:"column:cost_center" json:"cost_center"`
	SettlementDate         time.Time `gorm:"column:settlement_date" json:"settlement_date"`
	TransactionDate        time.Time `gorm:"column:transaction_date" json:"transaction_date"`
	Time                   string    `gorm:"column:time" json:"time"`
	TransactionDescription string    `gorm:"column:transaction_description" json:"transaction_description"`
	InvoiceNumber          string    `gorm:"column:invoice_number" json:"invoice_number"`
	Product                string    `gorm:"column:product" json:"product"`
	Liter                  float64   `gorm:"column:liter" json:"liter"`
	Price                  float64   `gorm:"column:price" json:"price"`
	AmountBeforeVAT        float64   `gorm:"column:amount_before_vat" json:"amount_before_vat"`
	VAT                    float64   `gorm:"column:vat" json:"vat"`
	TotalAmount            float64   `gorm:"column:total_amount" json:"total_amount"`
	WHT1Percent            float64   `gorm:"column:wht_1_percent" json:"wht_1_percent"`
	TotalAmountAfterWD     float64   `gorm:"column:total_amount_after_wd" json:"total_amount_after_wd"`
	Odometer               int       `gorm:"column:odometer" json:"odometer"`
	MerchantID             string    `gorm:"column:merchant_id" json:"merchant_id"`
	MerchantName           string    `gorm:"column:merchant_name" json:"merchant_name"`
	MerchantAccountTaxID   string    `gorm:"column:merchant_account_tax_id" json:"merchant_account_tax_id"`
	TaxBranch              string    `gorm:"column:tax_branch" json:"tax_branch"`
	Address                string    `gorm:"column:address" json:"address"`
	FuelBrand              string    `gorm:"column:fuel_brand" json:"fuel_brand"`
}

type EncryptedReport struct {
	ID        uint   `gorm:"primaryKey"`
	FileName  string `gorm:"not null"`
	FileData  []byte
	CreatedAt time.Time
}

func Connect(cfg *config.Config) *gorm.DB {
	dsn := "host=" + cfg.DbHost + " user=" + cfg.DbUser + " password=" + cfg.DbPass + " dbname=" + cfg.DbName + " port=" + cfg.DbPort + " sslmode=disable TimeZone=Asia/Bangkok"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	db.AutoMigrate(&Transaction{}, &EncryptedReport{})

	return db
}
