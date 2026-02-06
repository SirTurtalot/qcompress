# QCompress - Fast Encrypted Compression Tool

A high-performance file encryption and compression utility using AES-256 encryption and Zstandard compression.

## 🚀 Quick Start

```bash
# 1. Download Zastandard

go mod download github.com/klauspost/compress

# 2. Build the tool
go build -o qcompress qcompressstream_optimized.go

# 3. Generate a secure key
KEY=$(openssl rand -hex 32)

# 4. Encrypt a file
./qcompress -mode=encrypt -in=myfile.txt -out=myfile.enc -key=$KEY -level=3

# 5. Decrypt the file
./qcompress -mode=decrypt -in=myfile.enc -out=myfile.txt -key=$KEY
```

## 📦 What's Included

### Essential Files (4 files)

1. **qcompressstream_optimized.go** - Main program (optimized version)
2. **go.mod** - Go dependencies (required for building)
3. **create_realistic_files.sh** - Generate test files
4. **benchmark_realistic.sh** - Benchmark performance

## 🎯 Features

- ✅ **AES-256-CTR encryption** - Military-grade security
- ✅ **HMAC-SHA256 authentication** - Integrity verification
- ✅ **Zstandard compression** - 3-10x faster than gzip
- ✅ **Multi-threaded** - Uses all CPU cores
- ✅ **Configurable compression levels** - Balance speed vs size
- ✅ **Streaming operation** - Works with large files

## 📊 Performance Results

### Compression Ratios (Real Files)

| File Type        | Compression | Example                   |
| ---------------- | ----------- | ------------------------- |
| XML Config       | **99.4%**   | 10MB → 58KB               |
| Source Code      | **99.8%**   | 4.6MB → 9.5KB             |
| HTML Pages       | **97.4%**   | 8.3MB → 221KB             |
| JSON APIs        | **92.3%**   | 16MB → 1.2MB              |
| SQL Dumps        | **93.1%**   | 7.8MB → 552KB             |
| Server Logs      | **82.0%**   | 14MB → 2.5MB              |
| CSV Data         | **69.8%**   | 9.8MB → 3.0MB             |
| Plain Text       | **71.1%**   | 2.2MB → 645KB             |
| Binary/Encrypted | **0%**      | No compression (expected) |

### Speed Improvements vs Original

| File Size | Encryption  | Decryption   | Total Speedup |
| --------- | ----------- | ------------ | ------------- |
| 10MB      | 3-4x faster | 4-5x faster  | **~4x**       |
| 100MB     | 4-5x faster | 5-7x faster  | **~5x**       |
| 1GB       | 5-6x faster | 7-10x faster | **~7x**       |

## 🔧 Installation

### Prerequisites

- Go 1.21 or higher
- Git (for cloning)

### Build

```bash
# Download dependencies
go mod download

# Build the optimized version
go build -o qcompress qcompressstream_optimized.go

```

## 📖 Usage

### Basic Encryption/Decryption

```bash
# Generate a random key (save this!)
KEY=$(openssl rand -hex 32)
echo "Your key: $KEY"

# Encrypt
./qcompress -mode=encrypt -in=document.pdf -out=document.enc -key=$KEY

# Decrypt
./qcompress -mode=decrypt -in=document.enc -out=document.pdf -key=$KEY
```

### Compression Levels

```bash
# Level 1: Fastest (good for large files)
./qcompress -mode=encrypt -in=file.txt -out=file.enc -key=$KEY -level=1

# Level 2: Default (balanced)
./qcompress -mode=encrypt -in=file.txt -out=file.enc -key=$KEY -level=2

# Level 3: Better compression (recommended)
./qcompress -mode=encrypt -in=file.txt -out=file.enc -key=$KEY -level=3

# Level 4: Best compression (slower)
./qcompress -mode=encrypt -in=file.txt -out=file.enc -key=$KEY -level=4
```

## 🧪 Testing

### Generate Test Files

```bash
# Create 12 different file types (~10MB each)
./create_realistic_files.sh
```

This creates:

- **Highly compressible**: server.log, api_response.json, sales_data.csv, config.xml, article.txt, database.sql, source_code.js, webpage.html
- **Poorly compressible**: binary.bin, encrypted.dat, photo.jpg, archive.zip

### Run Benchmark

```bash
cd realistic_test_files
../benchmark_realistic.sh
```

This will test all file types and show:

- Original vs encrypted file sizes
- Compression ratios
- Encryption/decryption times
- Integrity verification

## 🔐 Security

- **Encryption**: AES-256 in CTR mode
- **Authentication**: HMAC-SHA256
- **IV**: Random 16-byte IV per file
- **Key**: 32-byte (256-bit) key required
- **Integrity**: Constant-time MAC comparison prevents timing attacks

## ⚠️ Key Management

- Never share your encryption key
- Store keys securely (password manager, key vault)
- Loss of key = permanent data loss
- Each file uses a unique IV (stored in encrypted file)

## 💡 When to Use Each Compression Level

| Level | Use Case                                 | Speed   | Ratio     |
| ----- | ---------------------------------------- | ------- | --------- |
| 1     | Large files (>1GB), real-time processing | Fastest | Good      |
| 2     | General purpose                          | Fast    | Better    |
| 3     | **Recommended** - Best balance           | Good    | Very Good |
| 4     | Archival storage, small files            | Slower  | Best      |

## 📁 Use Cases

Perfect for:

- 📝 **Log file archival** (80-95% compression)
- 💾 **Database backups** (75-90% compression)
- 🌐 **API response caching** (85-95% compression)
- 📊 **Data exports** (CSV, JSON, XML)
- 💻 **Source code backups** (65-80% compression)
- 🔒 **Secure file transfers**

Not recommended for:

- ❌ Already compressed files (ZIP, RAR, 7Z)
- ❌ Media files (JPG, PNG, MP4, MP3)
- ❌ Files where you need random access

## 🛠️ Technical Details

### How It Works

1. **Compression**: File is compressed using Zstandard
2. **Encryption**: Compressed data is encrypted with AES-256-CTR
3. **Authentication**: HMAC-SHA256 is computed over IV + ciphertext
4. **Output Format**: `[IV (16 bytes)] + [Encrypted Compressed Data] + [HMAC (32 bytes)]`

### Why Compress THEN Encrypt?

- Compression works best on unencrypted data (patterns)
- Encryption output is random (incompressible)
- Order matters: Compress → Encrypt (not Encrypt → Compress)

### Dependencies

- `github.com/klauspost/compress/zstd` - High-performance Zstandard implementation

## 🐛 Troubleshooting

## "Invalid 32-byte hex key"

- Key must be exactly 64 hex characters (32 bytes)
- Generate with: `openssl rand -hex 32`

## "Integrity check failed"

- Wrong decryption key
- File was modified/corrupted
- File is not an encrypted file

## Binary files not compressing

- This is expected! Random/encrypted data doesn't compress
- See benchmark results for examples

## Go build fails

- Run `go mod download` first
- Check Go version: `go version` (need 1.21+)
- Verify go.mod file exists

## 📄 License

This is a demonstration project. Use at your own risk.

## 🙏 Credits

- Zstandard compression by Facebook/Meta
- Go implementation by Klaus Post (klauspost/compress)

---

## Made with ❤️ for fast, secure file compression\*\*
