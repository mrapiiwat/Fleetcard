package services

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetcard/db"
)

func DecryptAndExtractCSV(report db.EncryptedReport, dateFormat string) ([]db.Transaction, error) {
	tmpDir := "./tmp"
	os.MkdirAll(tmpDir, os.ModePerm)

	// เขียนไฟล์ .gpg ชั่วคราว
	gpgPath := filepath.Join(tmpDir, report.FileName)
	if err := os.WriteFile(gpgPath, report.FileData, 0644); err != nil {
		return nil, fmt.Errorf("write gpg failed: %v", err)
	}

	// ถอดรหัส .gpg เป็น .zip
	zipPath := strings.TrimSuffix(gpgPath, ".gpg")
	err := exec.Command("gpg", "--batch", "--yes", "--output", zipPath, "--decrypt", gpgPath).Run()
	if err != nil {
		return nil, fmt.Errorf("decrypt gpg failed: %v", err)
	}

	// แตก zip
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("zip open failed: %v", err)
	}
	defer r.Close()

	var dataFilePath string
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".csv") || strings.HasSuffix(f.Name, ".txt") {
			dataFilePath = filepath.Join(tmpDir, f.Name)
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			os.MkdirAll(filepath.Dir(dataFilePath), os.ModePerm)

			out, err := os.Create(dataFilePath)
			if err != nil {
				return nil, err
			}
			defer out.Close()

			_, err = io.Copy(out, rc)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if dataFilePath == "" {
		return nil, errors.New("no .csv or .txt file found in zip")
	}

	// อ่านข้อมูลไฟล์ที่แยกออกมา และแปลงเป็น Transaction
	file, err := os.Open(dataFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var txs []db.Transaction
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if lineNum == 1 {
			continue // ข้าม header
		}

		row := strings.Split(line, "|")
		if len(row) < 26 {
			continue
		}

		settleDate, _ := time.Parse(dateFormat, row[6])
		transDate, _ := time.Parse(dateFormat, row[7])
		liter, _ := strconv.ParseFloat(row[12], 64)
		price, _ := strconv.ParseFloat(row[13], 64)
		amountBeforeVAT, _ := strconv.ParseFloat(row[14], 64)
		vat, _ := strconv.ParseFloat(row[15], 64)
		totalAmount, _ := strconv.ParseFloat(row[16], 64)
		wht, _ := strconv.ParseFloat(row[17], 64)
		afterWD, _ := strconv.ParseFloat(row[18], 64)
		odo, _ := strconv.Atoi(row[19])

		tx := db.Transaction{
			AccountNo:              row[0],
			AccountName:            row[1],
			FleetCardNumber:        row[2],
			LicensePlateNumber:     row[3],
			CardholderName:         row[4],
			CostCenter:             row[5],
			SettlementDate:         settleDate,
			TransactionDate:        transDate,
			Time:                   row[8],
			TransactionDescription: row[9],
			InvoiceNumber:          row[10],
			Product:                row[11],
			Liter:                  liter,
			Price:                  price,
			AmountBeforeVAT:        amountBeforeVAT,
			VAT:                    vat,
			TotalAmount:            totalAmount,
			WHT1Percent:            wht,
			TotalAmountAfterWD:     afterWD,
			Odometer:               odo,
			MerchantID:             row[20],
			MerchantName:           row[21],
			MerchantAccountTaxID:   row[22],
			TaxBranch:              row[23],
			Address:                row[24],
			FuelBrand:              row[25],
		}
		txs = append(txs, tx)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return txs, nil
}

func SaveTransactions(txs []db.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	if err := db.DB.CreateInBatches(txs, 100).Error; err != nil {
		return fmt.Errorf("failed to save transactions: %v", err)
	}

	return nil
}
