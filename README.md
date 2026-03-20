# QCompress - Fast Encrypted Compression Tool

A high-performance file encryption and compression utility using AES-256 encryption, Zstandard compression, and optional Argon2id password-based key derivation.

## 🚀 Quick Start

```bash
# 1. Download dependencies
go mod tidy

# 2. Build the tool
go build -o qcompress qcompressstream_optimized.go

# 3a. Encrypt with a password (easiest)
./qcompress -mode=encrypt -in=myfile.txt -out=myfile.enc -password="my secret passphrase"

# 3b. Or encrypt with a raw key (generated once and saved)
KEY=$(openssl rand -hex 32)
export QCOMPRESS_KEY=$KEY   # avoids key appearing in shell history
./qcompress -mode=encrypt -in=myfile.txt -out=myfile.enc

# 4. Decrypt
./qcompress -mode=decrypt -in=myfile.enc -out=myfile_recovered.txt -password="my secret passphrase"
# or with key:
./qcompress -mode=decrypt -in=myfile.enc -out=myfile_recovered.txt
```

---

## 📦 What's Included

| File                           | Purpose                  |
| ------------------------------ | ------------------------ |
| `qcompressstream_optimized.go` | Main program             |
| `go.mod`                       | Go module & dependencies |
| `create_realistic_files.sh`    | Generate test files      |
| `benchmark_realistic.sh`       | Benchmark performance    |

---

## 🎯 Features

- ✅ **AES-256-CTR encryption** — Fast, parallelizable, no padding oracle
- ✅ **HMAC-SHA256 authentication** — Authenticate-then-decrypt; constant-time comparison
- ✅ **Zstandard compression** — 3–10× faster than gzip with similar ratios
- ✅ **Multi-threaded** — Uses all CPU cores for both compression and decompression
- ✅ **Password mode** — Argon2id key derivation (OWASP-recommended parameters)
- ✅ **Raw key mode** — 32-byte key via flag, key file, or `QCOMPRESS_KEY` env var
- ✅ **Progress bar** — Live byte-rate progress for both encrypt and decrypt
- ✅ **Stdin/stdout support** — Encrypt from stdin; pipe-friendly output
- ✅ **Clean failure** — Partial output files are always deleted on error
- ✅ **Versioned file format** — Magic header enables future format evolution
- ✅ **fsync before exit** — Output is flushed to disk before the process reports success

---

## 🔑 Key Options

There are three ways to supply a key. **Choose exactly one per operation.**

### Option 1 — Password (Argon2id key derivation)

```bash
./qcompress -mode=encrypt -in=file.txt -out=file.enc -password="my passphrase"
./qcompress -mode=decrypt -in=file.enc -out=file.txt -password="my passphrase"
```

- A random 32-byte salt is generated at encryption time and stored in the file header.
- On decryption the salt is read from the header automatically — you only need the passphrase.
- Key derivation takes ~0.5 s (Argon2id, 64 MB memory). Progress is printed to stderr.
- **Best for**: human-facing use, backups, any situation where you don't want to manage a key file.

### Option 2 — Raw hex key via flag

```bash
KEY=$(openssl rand -hex 32)
./qcompress -mode=encrypt -in=file.txt -out=file.enc -key=$KEY
./qcompress -mode=decrypt -in=file.enc -out=file.txt -key=$KEY
```

> ⚠️ The key appears in shell history. Prefer `-keyfile` or `QCOMPRESS_KEY`.

### Option 3 — Env var (recommended over -key flag)

```bash
export QCOMPRESS_KEY=$(openssl rand -hex 32)
./qcompress -mode=encrypt -in=file.txt -out=file.enc
./qcompress -mode=decrypt -in=file.enc -out=file.txt
```

### Option 4 — Key file

```bash
# Generate a binary key file
openssl rand 32 > mykey.bin
chmod 600 mykey.bin

./qcompress -mode=encrypt -in=file.txt -out=file.enc -keyfile=mykey.bin
./qcompress -mode=decrypt -in=file.enc -out=file.txt -keyfile=mykey.bin
```

The key file may contain either 32 raw bytes or a 64-character hex string.

---

## 📖 Full Usage

```bash
Usage: qcompress -mode=encrypt|decrypt -in=<file> -out=<file> [key option] [-level=1-4]

Flags:
  -mode        encrypt or decrypt (required)
  -in          Input file path; use - for stdin (encrypt only)
  -out         Output file path; use - for stdout; default: stdout
  -key         32-byte key as 64 hex chars
  -keyfile     Path to key file (32 raw bytes or 64-char hex string)
  -password    Passphrase — Argon2id key derivation
  -level       Zstd compression level 1-4 (encrypt only; default 3)

Environment:
  QCOMPRESS_KEY   Hex key (equivalent to -key but not visible in shell history)
```

---

## 💡 Compression Levels

| Level | Speed    | Ratio         | Best For                           |
| ----- | -------- | ------------- | ---------------------------------- |
| 1     | Fastest  | Good          | Large files (>1 GB), real-time use |
| 2     | Fast     | Better        | General purpose                    |
| **3** | **Good** | **Very Good** | **Recommended (default)**          |
| 4     | Slower   | Best          | Archival storage, small files      |

---

## 📊 Performance Results

### Compression Ratios

| File Type        | Compression | Example                   |
| ---------------- | ----------- | ------------------------- |
| XML Config       | **99.4%**   | 10 MB → 58 KB             |
| Source Code      | **99.8%**   | 4.6 MB → 9.5 KB           |
| HTML Pages       | **97.4%**   | 8.3 MB → 221 KB           |
| JSON APIs        | **92.3%**   | 16 MB → 1.2 MB            |
| SQL Dumps        | **93.1%**   | 7.8 MB → 552 KB           |
| Server Logs      | **82.0%**   | 14 MB → 2.5 MB            |
| CSV Data         | **69.8%**   | 9.8 MB → 3.0 MB           |
| Plain Text       | **71.1%**   | 2.2 MB → 645 KB           |
| Binary/Encrypted | **0%**      | No compression (expected) |

### Speed vs Original gzip-based Version

| File Size | Encryption  | Decryption   | Total Speedup |
| --------- | ----------- | ------------ | ------------- |
| 10 MB     | 3–4× faster | 4–5× faster  | **~4×**       |
| 100 MB    | 4–5× faster | 5–7× faster  | **~5×**       |
| 1 GB      | 5–6× faster | 7–10× faster | **~7×**       |

---

## 🔐 Security

### Cryptographic Design

- **Encryption**: AES-256-CTR — parallelizable, no padding oracle vulnerability
- **Authentication**: HMAC-SHA256 — Encrypt-then-MAC construction, constant-time comparison
- **Key derivation**: Argon2id (1 pass, 64 MB memory, 4 threads, 32-byte output) — OWASP minimums
- **IV**: Random 16-byte IV generated per file; stored in header
- **Salt**: Random 32-byte salt generated per encryption (password mode); stored in header

### File Format (v1)

```bash
[Magic "QCS1" — 4 bytes]
[Flags — 1 byte: 0x00 = raw key, 0x01 = password/Argon2id]
[Argon2 Salt — 32 bytes, present only when flags = 0x01]
[IV — 16 bytes]
[AES-256-CTR( zstd( plaintext ) ) — variable length]
[HMAC-SHA256 — 32 bytes]
```

The HMAC covers **all bytes from the magic header through to the end of the ciphertext**, providing full authenticated encryption with associated data (AEAD) semantics. The MAC is verified before any decryption is attempted.

### Key Exposure Risk

| Method          | Shell History | `/proc` Exposure | Recommended              |
| --------------- | ------------- | ---------------- | ------------------------ |
| `-password`     | ⚠️ Yes        | ⚠️ Yes           | ✅ Okay for personal use |
| `-key`          | ❌ Exposed    | ❌ Exposed       | ⚠️ Avoid                 |
| `QCOMPRESS_KEY` | ✅ Safe       | ⚠️ Minor         | ✅ Good                  |
| `-keyfile`      | ✅ Safe       | ✅ Safe          | ✅ Best for automation   |

For production or CI/CD pipelines, use `-keyfile` or a secrets manager that injects `QCOMPRESS_KEY`.

---

## 🏗️ Installation

### Prerequisites

- Go 1.21 or higher
- Git

### Build

```bash
# Download dependencies
go mod tidy

# Build
go build -o qcompress qcompressstream_optimized.go

# Verify it works
echo "hello world" | ./qcompress -mode=encrypt -in=- -out=/tmp/test.enc -password="test"
./qcompress -mode=decrypt -in=/tmp/test.enc -out=- -password="test"
```

### Dependencies

| Package                              | Purpose                                |
| ------------------------------------ | -------------------------------------- |
| `github.com/klauspost/compress/zstd` | High-performance Zstandard compression |
| `github.com/schollz/progressbar/v3`  | Live progress bar output to stderr     |
| `golang.org/x/crypto/argon2`         | Argon2id key derivation                |

---

## 🧪 Testing

### Generate Test Files

```bash
./create_realistic_files.sh
```

Creates 12 different file types (~10 MB each):

- **Highly compressible**: server.log, api_response.json, sales_data.csv, config.xml, article.txt, database.sql, source_code.js, webpage.html
- **Poorly compressible**: binary.bin, encrypted.dat, photo.jpg, archive.zip

### Run Benchmark

```bash
../benchmark_realistic.sh
```

---

## 📁 Use Cases

**Great fits:**

- 📝 Log file archival (80–95% compression)
- 💾 Database backups (75–90% compression)
- 🌐 API response caching (85–95% compression)
- 📊 Data exports (CSV, JSON, XML)
- 💻 Source code backups
- 🔒 Secure file transfers

**Not recommended for:**

- ❌ Already compressed files (ZIP, RAR, 7Z, tar.gz)
- ❌ Media files (JPG, PNG, MP4, MP3)
- ❌ Files requiring random access or partial decryption

---

## 🛠️ Technical Details

### Processing Pipeline

```bash
Encrypt:  plaintext → zstd → AES-256-CTR ─┬──→ output file
                                            └──→ HMAC accumulator → append MAC

Decrypt:  Read + verify HMAC first
          → AES-256-CTR → zstd → plaintext output
```

Compression runs **before** encryption because encryption output is statistically random and incompressible. The HMAC is computed and verified **before** any decryption output is written, preventing decryption of tampered ciphertext.

### Improvements vs Original Version

| Area                  | Original                     | This Version                     |
| --------------------- | ---------------------------- | -------------------------------- |
| Key input             | CLI flag only (history risk) | Flag, file, env var, or password |
| Password support      | ❌ None                      | ✅ Argon2id derivation           |
| Failed output cleanup | ❌ Partial file left         | ✅ Deleted on any error          |
| Progress reporting    | ❌ Silent                    | ✅ Live progress bar             |
| File format           | No header                    | ✅ Magic + versioned flags       |
| Disk flush            | ❌ No fsync                  | ✅ fsync before exit             |
| Stdin support         | ❌ No                        | ✅ `-in=-` for encrypt           |
| Error messages        | Generic                      | ✅ Actionable descriptions       |

---

## 🐛 Troubleshooting

### "not a QCompress file (invalid magic)"

The file was not encrypted by qcompress, or is corrupted. It may have been created by the old version of this tool (which had no file header) — in that case, re-encrypt using the old binary.

### "file was encrypted with a passphrase; provide -password"

The `.enc` file was encrypted in password mode. Supply `-password=` instead of `-key=`.

### "file was encrypted with a raw key; use -key / -keyfile"

The `.enc` file was encrypted in raw key mode. Supply `-key=`, `-keyfile=`, or `QCOMPRESS_KEY`.

### "integrity check failed"

Either the wrong key/password was supplied, or the file has been modified or corrupted since encryption. Do not trust its contents.

### "Invalid 32-byte hex key"

The value of `-key` or `QCOMPRESS_KEY` must be exactly 64 hexadecimal characters (representing 32 bytes). Generate one with `openssl rand -hex 32`.

### Binary files not compressing

Expected. Random and already-compressed data has no redundancy for zstd to exploit. See the performance table above.

### `go mod tidy` fails

- Verify your Go version: `go version` (needs 1.21+)
- Ensure internet access to `proxy.golang.org`
- Try `GOPROXY=direct go mod tidy`

---

## ⚠️ Key Management

- **Never lose your key or passphrase** — there is no recovery mechanism
- Store keys in a password manager or key vault (e.g. 1Password, HashiCorp Vault, AWS Secrets Manager)
- Each file gets a unique random IV (and unique salt in password mode); encrypted copies of the same file will differ
- For long-term archival, store the key alongside a printed copy of this README so future-you knows the tool and format

---

## 📄 License

Demonstration project. Use at your own risk.

## 🙏 Credits

- Zstandard by Facebook/Meta; Go implementation by Klaus Post (`klauspost/compress`)
- Progress bar by schollz (`schollz/progressbar`)
- Argon2 by the Password Hashing Competition; Go wrapper by the Go team (`golang.org/x/crypto`)

---

## Made with ❤️ for fast, secure file compression
