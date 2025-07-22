# Fleetcard Manager

Fleetcard Manager is a backend service built with Go and Echo framework, designed to securely receive, decrypt, and store daily fleet card transaction reports sent from a bank via SFTP.

## Features

- Connects to SFTP server and downloads `.gpg` encrypted report files.
- Decrypts `.gpg` files using GPG and extracts `.zip` contents.
- Supports extracting `.csv` or `.txt` transaction data inside the archive.
- Parses `|`-delimited data and maps it into `Transaction` struct (26 fields).
- Saves all parsed records into a PostgreSQL database using GORM.
- Reads `DATE_FORMAT` from `.env` file for flexible date parsing.

## Folder Structure

Fleetcard/
├── cmd/ # Entry point (main.go)\n
├── config/ # Load .env and app config\n
├── controllers/ # Download from SFTP\n
├── db/ # Database models and connection\n
├── services/ # Decrypt, extract, and parse logic\n
├── .env # Environment variables\n
├── docker-compose.yml\n
├── go.mod / go.sum\n

## How to Run

1. Install [GPG](https://gnupg.org) and import your private key.
2. Put `.gpg` files in the SFTP server folder.
3. Fill `.env` file with credentials and config:

DB_HOST=localhost
DB_PORT=15432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=fleetcarddb
DATE_FORMAT=02/01/2006

SFTP_HOST=...
SFTP_PORT=22
SFTP_USER=...
SFTP_PASSWORD=...
SFTP_REMOTE_DIR=/fleetcard

4. Run the app: go run ./cmd/main.go

# Tech Stack

- Language: Go
- Framework: Echo
- ORM: GORM
- Database: PostgreSQL
- File Transfer: SFTP
- Decryption: GPG
