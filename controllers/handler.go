package controllers

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fleetcard/services"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// เชื่อมไปเซิร์ฟเวอร์ SFTP, loops หา .gpg files ใน /inbound,
func ProcessAllInboundFiles(dateFormat string) {
	sftpHost := os.Getenv("SFTP_HOST")
	sftpPort := os.Getenv("SFTP_PORT")
	sftpUser := os.Getenv("SFTP_USER")
	sftpPassword := os.Getenv("SFTP_PASSWORD")
	remoteInbound := os.Getenv("SFTP_REMOTE_INBOUND_DIR")

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

	files, err := client.ReadDir(remoteInbound)
	if err != nil {
		log.Fatalf("Failed to read dir %s: %v", remoteInbound, err)
	}

	found := 0

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".gpg") {
			log.Printf("Found file: %s", file.Name())

			err := services.DecryptAndExtract(file.Name(), dateFormat)
			if err != nil {
				log.Printf("Failed: %s → %v", file.Name(), err)
			} else {
				log.Printf("Success: %s", file.Name())
			}

			time.Sleep(1 * time.Second)
		}
	}
	if found == 0 {
		log.Printf("No .gpg files found in directory: %s", remoteInbound)
	}
}
