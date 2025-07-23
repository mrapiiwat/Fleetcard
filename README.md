# Fleetcard Manager  
  
ระบบ Fleetcard Manager คือระบบ backend ที่ใช้สำหรับเชื่อมต่อกับ SFTP server เพื่อดึงข้อมูลรายงานการใช้งานบัตร Fleetcard (ในรูปแบบ `.gpg`) มาถอดรหัสและบันทึกข้อมูลธุรกรรมลงในฐานข้อมูล PostgreSQL โดยอัตโนมัติ  
  
---  
  
## Features  
  
- เชื่อมต่อกับ **SFTP Server** เพื่อดึงไฟล์ `.gpg` จากโฟลเดอร์ `/inbound`  
- ถอดรหัสไฟล์ `.gpg` และแตก `.zip` ภายในเพื่อดึงไฟล์ `.txt` หรือ `.csv`  
- แปลงข้อมูลธุรกรรมเป็น **26 คอลัมน์** ตามโครงสร้างของระบบ  
- บันทึกข้อมูลลง **ฐานข้อมูล PostgreSQL** ด้วย GORM  
- เมื่อเสร็จสิ้น จะย้ายไฟล์ `.gpg` ไปยังโฟลเดอร์ `/outbound`  
- รองรับการใช้ **GPG passphrase** ผ่านตัวแปร environment เพื่อความปลอดภัย  

---
  
## วิธีใช้งาน  
### 1. ติดตั้งระบบที่ใช้ถอดรหัส .gpg และ insert private key เข้าไปยัง device
### 2. โคลนโปรเจกต์
git clone https://github.com/mrapiiwat/Fleetcard.git  
### 3. สร้างไฟล์ .env
DB_HOST=localhost  
DB_PORT=15432  
DB_USER=postgres  
DB_PASSWORD=your_password  
DB_NAME=fleetcarddb  
  
DATE_FORMAT=02/01/2006  
  
GPG_PASSPHRASE=your_passphrase  
  
SFTP_HOST=your_sftp_host  
SFTP_PORT=22  
SFTP_USER=your_user  
SFTP_PASSWORD=your_password  
SFTP_REMOTE_INBOUND_DIR=/inbound  
SFTP_REMOTE_OUTBOUND_DIR=/outbound  
### 4. รัน PostgreSQL ด้วย Docker
docker-compose up -d  
### 5. สร้าง directory ใน SFTP server (สำหรับ test)
- inbound
- outbound  
ไฟล์ .gpg ที่ต้องการถอดรหัสจะต้องอยู่ใน inbound  
### 6. Run
go run cmd/main.go  
ระบบจะทำการ:  
- เชื่อมต่อกับ SFTP server
- ดึงและถอดรหัสไฟล์ .gpg
- บันทึกธุรกรรมลงในฐานข้อมูล
- ย้ายไฟล์ไปยัง /outbound บน SFTP server
