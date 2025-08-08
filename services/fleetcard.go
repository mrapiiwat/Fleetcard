package services

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/fleetcard/db"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// อ่าน Private Key + ถอดรหัสไฟล์ .gpg ไปเป็น .zip ด้วย ProtonMail
func decryptWithPrivateKey(inputPath, outputPath, privateKeyPath, passphrase string) error {
	// อ่าน encrypted file
	encryptedData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("cannot read encrypted file: %v", err)
	}
	inputReader := bytes.NewReader(encryptedData)

	// อ่าน private key
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("cannot read private key: %v", err)
	}

	// สร้าง entityList
	var entityList openpgp.EntityList
	if bytes.HasPrefix(keyData, []byte("-----BEGIN")) {
		entityList, err = openpgp.ReadArmoredKeyRing(bytes.NewReader(keyData))
	} else {
		entityList, err = openpgp.ReadKeyRing(bytes.NewReader(keyData))
	}
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	// ปลดล็อกด้วย passphrase
	for _, entity := range entityList {
		if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
			if err := entity.PrivateKey.Decrypt([]byte(passphrase)); err != nil {
				return fmt.Errorf("failed to decrypt private key: %v", err)
			}
		}
		for _, sub := range entity.Subkeys {
			if sub.PrivateKey != nil && sub.PrivateKey.Encrypted {
				if err := sub.PrivateKey.Decrypt([]byte(passphrase)); err != nil {
					return fmt.Errorf("failed to decrypt subkey: %v", err)
				}
			}
		}
	}

	// detect armored
	var messageReader io.Reader = inputReader
	if bytes.HasPrefix(encryptedData, []byte("-----BEGIN")) {
		block, err := armor.Decode(inputReader)
		if err != nil {
			return fmt.Errorf("armor decode failed: %v", err)
		}
		messageReader = block.Body
	}

	// decrypt
	md, err := openpgp.ReadMessage(messageReader, entityList, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to read encrypted message: %v", err)
	}

	// เขียน decrypt output
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("cannot create output file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, md.UnverifiedBody)
	if err != nil {
		return fmt.Errorf("failed to write decrypted file: %v", err)
	}

	return nil
}

func DecryptAndExtract(fileName, dateFormat string) error {
	sftpHost := os.Getenv("SFTP_HOST")
	sftpPort := os.Getenv("SFTP_PORT")
	sftpUser := os.Getenv("SFTP_USER")
	sftpPassword := os.Getenv("SFTP_PASSWORD")
	remoteInbound := os.Getenv("SFTP_REMOTE_INBOUND_DIR")
	remoteOutbound := os.Getenv("SFTP_REMOTE_OUTBOUND_DIR")

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

	remotePath := path.Join(remoteInbound, fileName)
	localTmp := "./tmp"
	os.MkdirAll(localTmp, os.ModePerm)
	localGpgPath := path.Join(localTmp, fileName)

	// ดาวน์โหลด .gpg
	srcFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("cannot open remote file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localGpgPath)
	if err != nil {
		return fmt.Errorf("cannot create local file: %v", err)
	}
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return fmt.Errorf("failed to copy .gpg: %v", err)
	}
	dstFile.Close()

	// ถอดรหัส
	localZipPath := strings.TrimSuffix(localGpgPath, ".gpg")
	passphrase := os.Getenv("GPG_PASSPHRASE")
	fmt.Println("PASS:", os.Getenv("GPG_PASSPHRASE"))
	privateKeyPath := os.Getenv("GPG_PRIVATE_KEY_PATH")

	err = decryptWithPrivateKey(localGpgPath, localZipPath, privateKeyPath, passphrase)
	if err != nil {
		return fmt.Errorf("GPG decrypt failed: %v", err)
	}

	// แตก zip
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

			rc, err := f.Open()
			if err != nil {
				return err
			}

			out, err := os.Create(txtPath)
			if err != nil {
				rc.Close()
				return err
			}

			_, err = io.Copy(out, rc)

			rc.Close()
			out.Close()

			if err != nil {
				return err
			}
			break

		}
	}
	if txtPath == "" {
		return fmt.Errorf("no .txt/.csv file found in zip")
	}

	// อ่านไฟล์
	f, err := os.Open(txtPath)
	if err != nil {
		return err
	}

	var txs []db.Transaction
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum == 1 {
			continue
		}
		row := strings.Split(scanner.Text(), "|")
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

		txs = append(txs, db.Transaction{
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
		})
	}
	if err := scanner.Err(); err != nil {
		f.Close()
		return err
	}
	// ปิด txt
	f.Close()
	// ปิด zip
	r.Close()

	if len(txs) > 0 {
		if err := db.DB.CreateInBatches(txs, 100).Error; err != nil {
			return fmt.Errorf("failed to save transactions: %v", err)
		}
	}

	// ย้าย .gpg ไป outbound
	if err := client.Rename(remotePath, path.Join(remoteOutbound, fileName)); err != nil {
		return fmt.Errorf("failed to move file: %v", err)
	}

	time.Sleep(1 * time.Second)

	if err := os.RemoveAll(localTmp); err != nil {
		return fmt.Errorf("failed to clean up tmp directory: %v", err)
	}

	return nil
}
