# CINEMBROT - Golang Movie Scraper, Web Streaming & MariaDB

Aplikasi lengkap (*All-in-One*) portal film & web scraper performa tinggi berbasis Golang yang terhubung ke database **MariaDB**, dilengkapi:
- **Auto-Scraping Berkala (*Cron Background Scheduler*)**: Mengambil seluruh film secara otomatis lintas tahun secara berkala tanpa intervensi manual.
- **Mode Ramah Server (*Polite Rate Limiting & Anti-Ban*)**: Jeda waktu acak (*randomized jitter delay* 2.5–3.5 detik per film) dan batas *concurrency* rendah sehingga tidak membebani ataupun diblokir server target.
- **Web Streaming & Download Portal**: Mendukung **Light Mode (Default)** dan **Dark Mode** dengan Tailwind CSS.
- **Slot Iklan Siap Pakai untuk Monetisasi Adsterra** (Popunder, Social Bar, Header Banner 728x90, Native Banner di bawah player video).
- **Penandaan Lisensi & Legalitas** (*Public Domain, Creative Commons, Copyrighted Commercial*).

---

## 🤖 1. Cara Kerja Auto-Scraping Ramah Server (*Polite Auto-Scraper*)

Fitur auto-scraping otomatis berjalan di background saat web server aktif (`go run main.go -serve`).

### 🛡️ Fitur Proteksi Server Target:
1. **Jeda Waktu (*Polite Delay & Jitter*)**:
   - Memberi jeda `2.5 detik + random 0–1 detik` antar request film.
   - Server target mendeteksi aktivitas browsing alami (*human-like*), bukan serangan brute-force / DDoS.
2. **Rotasi Kunci API (*Key Pool Rotation*)**:
   - Berputar otomatis di antara puluhan TMDb/OMDb API keys sehingga tidak pernah terkena limit kuota.
3. **Penelusuran Seluruh Tahun (*Year Sweep*)**:
   - Otomatis menelusuri film dari tahun terbaru (`2024`) turun hingga tahun klasik (`1990`).
   - Mengambil film-film Creative Commons & film legal dari Internet Archive.

---

## ⚙️ 2. Pengaturan Auto-Scraping di `.env`

Anda dapat mengatur frekuensi dan jeda waktu di file `.env`:

```ini
# Aktifkan auto-scraping otomatis
AUTO_SCRAPE_ENABLED=true

# Jadwal scraping berkala (dalam menit, default: 360 = setiap 6 jam)
AUTO_SCRAPE_INTERVAL_MINUTES=360

# Jeda ramah antar request film (dalam milidetik, default: 2500ms = 2.5 detik)
AUTO_SCRAPE_DELAY_MS=2500

# Rentang tahun yang ingin ditelusuri
AUTO_SCRAPE_START_YEAR=2024
AUTO_SCRAPE_END_YEAR=1990
AUTO_SCRAPE_PAGES_PER_YEAR=2
```

---

## 🚀 3. Perintah Menjalankan Aplikasi

```powershell
# 1. Jalankan Web Server + Auto-Scraper Background (Rekomendasi):
go run main.go -serve

# 2. Jalankan 1 siklus scraping lengkap semua tahun sekarang juga:
go run main.go -auto-scrape

# 3. Jalankan auto-scraper sebagai background worker saja (tanpa web):
go run main.go -daemon

# 4. Scrape manual tahun tertentu:
go run main.go -by-year 2024 -pages 3
```

---

## 🌐 4. Akses Web Portal

Buka browser Anda di:
👉 **`http://localhost:8080`**
