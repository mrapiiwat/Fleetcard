package controllers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fleetcard/db"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func DownloadAllGPGFromSFTP() {
	sftpHost := os.Getenv("SFTP_HOST")
	sftpPort := os.Getenv("SFTP_PORT")
	sftpUser := os.Getenv("SFTP_USER")
	sftpPassword := os.Getenv("SFTP_PASSWORD")
	remoteDir := os.Getenv("SFTP_REMOTE_DIR")

	config := &ssh.ClientConfig{
		User: sftpUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sftpPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sftpHost, sftpPort), config)
	if err != nil {
		log.Fatalf("SSH Dial error: %v", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatalf("SFTP client error: %v", err)
	}
	defer client.Close()

	files, err := client.ReadDir(remoteDir)
	if err != nil {
		log.Fatalf("Failed to read dir %s: %v", remoteDir, err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".gpg") {
			fullPath := remoteDir + "/" + file.Name()
			fmt.Println("Downloading:", fullPath)

			var existing db.EncryptedReport
			if err := db.DB.Where("file_name = ?", file.Name()).First(&existing).Error; err == nil {
				fmt.Println("Skip already existing:", file.Name())
				continue
			}

			remoteFile, err := client.Open(fullPath)
			if err != nil {
				log.Println("Open file error:", err)
				continue
			}

			var buf bytes.Buffer
			_, err = io.Copy(&buf, remoteFile)
			remoteFile.Close()
			if err != nil {
				log.Println("Read file error:", err)
				continue
			}

			report := db.EncryptedReport{
				FileName:  file.Name(),
				FileData:  buf.Bytes(),
				CreatedAt: time.Now(),
			}

			result := db.DB.Create(&report)
			if result.Error != nil {
				log.Println("Insert DB error:", result.Error)
			} else {
				fmt.Println("Inserted:", file.Name())
			}
		}
	}
}
