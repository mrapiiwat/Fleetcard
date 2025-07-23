package services

import (
	"archive/zip"
	"bufio"
	"fmt"
	"github.com/fleetcard/db"
	"github.com/joho/godotenv"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

func DecryptAndExtract(fileName, dateFormat string) error {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	sftpHost := os.Getenv("SFTP_HOST")
	sftpPort := os.Getenv("SFTP_PORT")
	sftpUser := os.Getenv("SFTP_USER")
	sftpPassword := os.Getenv("SFTP_PASSWORD")
	remoteInbound := os.Getenv("SFTP_REMOTE_INBOUND_DIR")
	remoteOutbound := os.Getenv("SFTP_REMOTE_OUTBOUND_DIR")

	// Connect SFTP
	config := &ssh.ClientConfig{
		User:            sftpUser,
		Auth:            []ssh.AuthMethod{ssh.Password(sftpPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sftpHost, sftpPort), config)
	if err != nil {
		return fmt.Errorf("SSH dial failed: %v", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("SFTP client failed: %v", err)
	}
	defer client.Close()

	// โหลด .gpg จาก inbound
	remotePath := path.Join(remoteInbound, fileName)
	localTmp := "./tmp"
	os.MkdirAll(localTmp, os.ModePerm)
	localGpgPath := path.Join(localTmp, fileName)

	srcFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("cannot open remote file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localGpgPath)
	if err != nil {
		return fmt.Errorf("cannot create local file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to download .gpg: %v", err)
	}

	dstFile.Close()
	os.Chmod(localGpgPath, 0644)

	// Decrypt .gpg เป็น .zip
	localZipPath := strings.TrimSuffix(localGpgPath, ".gpg")

	passphrase := os.Getenv("GPG_PASSPHRASE")

	if passphrase == "" {
		return fmt.Errorf("GPG_PASSPHRASE is not set")
	}

	cmd := exec.Command(
		"gpg",
		"--batch",
		"--yes",
		"--passphrase", passphrase,
		"--pinentry-mode", "loopback",
		"--output", localZipPath,
		"--decrypt", localGpgPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GPG decrypt failed: %v\nOutput: %s", err, string(output))
	}

	// Unzip และอ่าน .txt
	r, err := zip.OpenReader(localZipPath)
	if err != nil {
		return fmt.Errorf("zip open failed: %v", err)
	}
	defer r.Close()

	var txtPath string
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".txt") || strings.HasSuffix(f.Name, ".csv") {
			txtPath = path.Join(localTmp, f.Name)
			os.MkdirAll(path.Dir(txtPath), os.ModePerm)

			rc, _ := f.Open()
			defer rc.Close()
			out, _ := os.Create(txtPath)
			defer out.Close()
			io.Copy(out, rc)
			break
		}
	}
	if txtPath == "" {
		return fmt.Errorf("no .txt/.csv file found in zip")
	}

	// Parse txt []Transaction
	file, err := os.Open(txtPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var txs []db.Transaction
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if lineNum == 1 {
			continue
		}
		row := strings.Split(line, "|")
		if len(row) < 26 {
			continue
		}

		settleDate, err := time.Parse(dateFormat, row[6])
		if err != nil {
			continue
		}
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
		return err
	}

	// บันทึกไปยังฐานข้อมูล
	if len(txs) > 0 {
		if err := db.DB.CreateInBatches(txs, 100).Error; err != nil {
			return fmt.Errorf("failed to save transactions: %v", err)
		}
	}

	// ย้าย .gpg ไป /outbound
	remoteDest := path.Join(remoteOutbound, fileName)
	if err := client.Rename(remotePath, remoteDest); err != nil {
		return fmt.Errorf("failed to move file to outbound: %v", err)
	}

	return nil
}
