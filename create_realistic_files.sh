#!/bin/bash

# Generate realistic test files for compression benchmarking (Windows Git Bash compatible)
# All files are approximately 10MB with proper binary/random data

echo "=================================================================="
echo "Creating Realistic Test Files for Compression (All ~10MB)"
echo "=================================================================="
echo ""

# Create output directory
mkdir -p realistic_test_files
cd realistic_test_files

# Helper function to get file size
get_size() {
    if [ -f "$1" ]; then
        wc -c < "$1" 2>/dev/null || echo "0"
    else
        echo "0"
    fi
}

# Helper function to generate random binary data
generate_random_binary() {
    local filename=$1
    local size_mb=$2
    local size_bytes=$((size_mb * 1024 * 1024))
    
    echo "Creating $filename (${size_mb}MB)..."
    
    # Method 1: Try openssl (best - truly random)
    if command -v openssl >/dev/null 2>&1; then
        openssl rand $size_bytes > "$filename" 2>/dev/null
        if [ -f "$filename" ] && [ $(get_size "$filename") -gt $((size_bytes - 1000)) ]; then
            echo "✓ $filename created using openssl ($(get_size "$filename") bytes)"
            return 0
        fi
    fi
    
    # Method 2: Try Python (good - uses os.urandom)
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import os; open('$filename', 'wb').write(os.urandom($size_bytes))" 2>/dev/null
        if [ -f "$filename" ] && [ $(get_size "$filename") -gt $((size_bytes - 1000)) ]; then
            echo "✓ $filename created using Python3 ($(get_size "$filename") bytes)"
            return 0
        fi
    fi
    
    if command -v python >/dev/null 2>&1; then
        python -c "import os; open('$filename', 'wb').write(os.urandom($size_bytes))" 2>/dev/null
        if [ -f "$filename" ] && [ $(get_size "$filename") -gt $((size_bytes - 1000)) ]; then
            echo "✓ $filename created using Python ($(get_size "$filename") bytes)"
            return 0
        fi
    fi
    
    # Method 3: Fallback - less ideal but better than nothing
    echo "⚠ Using fallback method (openssl/python not found) - may be slightly compressible"
    {
        for i in $(seq 1 $size_mb); do
            printf '%0999999d' $RANDOM | head -c 1048576
        done
    } > "$filename"
    echo "✓ $filename created using fallback ($(get_size "$filename") bytes)"
}

# 1. SERVER LOG FILE - Target: 10MB
echo "Creating server.log (10MB)..."
{
    IPS=("192.168.1.100" "10.0.0.45" "172.16.0.89" "192.168.1.101" "10.0.0.46")
    METHODS=("GET" "POST" "PUT" "DELETE")
    PATHS=("/api/users" "/api/products" "/api/orders" "/index.html" "/static/app.js" "/images/logo.png")
    STATUSES=(200 200 200 201 304 400 404 500)
    AGENTS=("Mozilla/5.0" "Chrome/120.0" "Safari/17.0" "Edge/119.0")
    
    for i in {1..150000}; do
        printf "2025-02-%02d %02d:%02d:%02d %s - - [%s %s HTTP/1.1] %d %d \"%s\" %dms\n" \
            $((RANDOM % 28 + 1)) $((RANDOM % 24)) $((RANDOM % 60)) $((RANDOM % 60)) \
            "${IPS[$RANDOM % 5]}" "${METHODS[$RANDOM % 4]}" "${PATHS[$RANDOM % 6]}" \
            "${STATUSES[$RANDOM % 8]}" $((RANDOM % 50000 + 100)) \
            "${AGENTS[$RANDOM % 4]}" $((RANDOM % 5000 + 10))
    done
} > server.log
echo "✓ server.log created ($(get_size server.log) bytes)"

# 2. JSON API RESPONSE - Target: 10MB
echo "Creating api_response.json (10MB)..."
{
    echo '{"status":"success","timestamp":"2025-02-05T14:30:00Z","data":['
    for i in {1..60000}; do
        printf '{"id":%d,"user_id":%d,"username":"user_%d","email":"user%d@example.com","status":"active","created_at":"2025-02-05T12:00:00Z","metadata":{"login_count":%d,"last_ip":"192.168.%d.%d","preferences":{"theme":"dark","notifications":true,"language":"en"}}}' \
            $i $((RANDOM % 10000)) $((RANDOM % 10000)) $((RANDOM % 10000)) \
            $((RANDOM % 1000)) $((RANDOM % 256)) $((RANDOM % 256))
        [ $i -lt 60000 ] && echo "," || echo ""
    done
    echo '],"total":60000,"page":1,"per_page":60000}'
} > api_response.json
echo "✓ api_response.json created ($(get_size api_response.json) bytes)"

# 3. CSV FILE - Target: 10MB
echo "Creating sales_data.csv (10MB)..."
{
    echo "transaction_id,date,customer_id,product_id,product_name,quantity,unit_price,total,payment_method,store_location"
    PRODUCTS=("Laptop" "Mouse" "Keyboard" "Monitor" "Headphones" "Webcam" "USB Cable")
    PAYMENT=("Credit Card" "Debit Card" "PayPal" "Cash")
    STORE=("New York" "Los Angeles" "Chicago" "Houston" "Phoenix")
    
    for i in {1..150000}; do
        printf "%d,2024-%02d-%02d,%d,%d,%s,%d,%d.%02d,%d.%02d,%s,%s\n" \
            $i $((RANDOM % 12 + 1)) $((RANDOM % 28 + 1)) $((RANDOM % 5000)) $((RANDOM % 100)) \
            "${PRODUCTS[$RANDOM % 7]}" $((RANDOM % 10 + 1)) $((RANDOM % 1000 + 10)) \
            $((RANDOM % 100)) $((RANDOM % 10000)) $((RANDOM % 100)) \
            "${PAYMENT[$RANDOM % 4]}" "${STORE[$RANDOM % 5]}"
    done
} > sales_data.csv
echo "✓ sales_data.csv created ($(get_size sales_data.csv) bytes)"

# 4. XML CONFIGURATION - Target: 10MB
echo "Creating config.xml (10MB)..."
{
    echo '<?xml version="1.0" encoding="UTF-8"?>'
    echo '<configuration><application name="WebApp" version="2.0.1">'
    for i in {1..30000}; do
        printf '<module id="module_%d"><n>Module %d</n><enabled>true</enabled><settings><setting key="timeout" value="30000"/><setting key="retries" value="3"/><setting key="cache_enabled" value="true"/><setting key="log_level" value="INFO"/></settings><dependencies><dependency>core</dependency><dependency>utils</dependency></dependencies></module>\n' $i $i
    done
    echo '</application></configuration>'
} > config.xml
echo "✓ config.xml created ($(get_size config.xml) bytes)"

# 5. PLAIN TEXT - Target: 10MB
echo "Creating article.txt (10MB)..."
{
    WORDS=("the" "be" "to" "of" "and" "a" "in" "that" "have" "I" "it" "for" "not" "on" "with" "he" "as" "you" "do" "at" "this" "but" "his" "by" "from" "they" "we" "say" "her" "she" "or" "an" "will" "my" "one" "all" "would" "there" "their" "when" "who" "which" "make" "can" "like" "time" "just" "know" "take" "people" "into" "year" "your" "good" "some" "could" "them" "see" "other" "than" "then" "now" "look" "only" "come" "its" "over" "think" "also" "back" "after" "use" "two" "how" "our" "work" "first" "well" "way" "even" "new" "want" "because" "any" "these" "give" "day" "most" "us")
    
    for chapter in {1..500}; do
        echo ""
        echo "CHAPTER $chapter"
        echo "============================================"
        echo ""
        for para in {1..40}; do
            sentence=""
            for word in {1..25}; do
                sentence="$sentence ${WORDS[$RANDOM % 90]}"
            done
            echo "$sentence."
        done
    done
} > article.txt
echo "✓ article.txt created ($(get_size article.txt) bytes)"

# 6. SQL DUMP - Target: 10MB
echo "Creating database.sql (10MB)..."
{
    echo "-- Database dump created on 2025-02-05 14:30:00"
    echo "-- Database: production_db"
    echo "CREATE TABLE IF NOT EXISTS users (id INT PRIMARY KEY AUTO_INCREMENT, username VARCHAR(255) NOT NULL, email VARCHAR(255) NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"
    echo ""
    for i in {1..100000}; do
        printf "INSERT INTO users (username, email) VALUES ('user_%d', 'user%d@example.com');\n" \
            $((RANDOM % 10000)) $((RANDOM % 10000))
    done
    echo "-- End of dump"
} > database.sql
echo "✓ database.sql created ($(get_size database.sql) bytes)"

# 7. SOURCE CODE - Target: 10MB
echo "Creating source_code.js (10MB)..."
{
    for file_num in {1..5000}; do
        echo "// Module: UserAuthentication - File $file_num"
        echo "const express = require('express');"
        echo "const jwt = require('jsonwebtoken');"
        echo "const bcrypt = require('bcrypt');"
        echo "class UserAuthenticationService {"
        echo "    constructor(config) { this.config = config; this.secretKey = process.env.JWT_SECRET || 'default-secret-key'; }"
        echo "    async authenticateUser(username, password) {"
        echo "        const user = await this.findUserByUsername(username);"
        echo "        if (!user) { throw new Error('User not found'); }"
        echo "        const isValidPassword = await bcrypt.compare(password, user.passwordHash);"
        echo "        if (!isValidPassword) { throw new Error('Invalid password'); }"
        echo "        const token = jwt.sign({ userId: user.id, username: user.username }, this.secretKey, { expiresIn: '24h' });"
        echo "        return { success: true, token, user };"
        echo "    }"
        echo "    async findUserByUsername(username) { return await database.query('SELECT * FROM users WHERE username = ?', [username]); }"
        echo "}"
        echo "module.exports = UserAuthenticationService;"
        echo ""
    done
} > source_code.js
echo "✓ source_code.js created ($(get_size source_code.js) bytes)"

# 8. HTML FILE - Target: 10MB
echo "Creating webpage.html (10MB)..."
{
    echo '<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Product Catalog</title><style>.product-card{border:1px solid #ccc;padding:20px;margin:10px;}.product-title{font-size:18px;font-weight:bold;}.product-price{color:#007bff;}</style></head><body><div class="container">'
    for i in {1..30000}; do
        printf '<div class="product-card"><h3 class="product-title">Product #%d</h3><p class="product-description">This is a detailed description of product number %d with features and benefits that make it great.</p><p class="product-price">$%d.99</p><button class="btn">Add to Cart</button></div>\n' $i $i $((RANDOM % 1000 + 10))
    done
    echo '</div></body></html>'
} > webpage.html
echo "✓ webpage.html created ($(get_size webpage.html) bytes)"

# 9. BINARY FILE - Target: 10MB (using generate_random_binary function)
generate_random_binary "binary.bin" 10

# 10. ENCRYPTED DATA - Target: 10MB (using generate_random_binary function)
generate_random_binary "encrypted.dat" 10

# 11. JPEG IMAGE - Target: 10MB
echo "Creating photo.jpg (10MB)..."
{
    # JPEG header
    printf '\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00'
    
    # Random data for image content
    if command -v openssl >/dev/null 2>&1; then
        openssl rand 10485700 2>/dev/null
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c "import os; import sys; sys.stdout.buffer.write(os.urandom(10485700))" 2>/dev/null
    elif command -v python >/dev/null 2>&1; then
        python -c "import os; import sys; sys.stdout.buffer.write(os.urandom(10485700))" 2>/dev/null
    else
        echo "⚠ Using fallback for photo.jpg"
        for i in $(seq 1 10); do
            printf '%0999999d' $RANDOM | head -c 1048570
        done
    fi
    
    # JPEG footer
    printf '\xFF\xD9'
} > photo.jpg
echo "✓ photo.jpg created ($(get_size photo.jpg) bytes)"

# 12. ZIP ARCHIVE - Target: 10MB
echo "Creating archive.zip (10MB)..."
{
    # ZIP header
    printf 'PK\x03\x04\x14\x00\x00\x00\x08\x00'
    
    # Random data
    if command -v openssl >/dev/null 2>&1; then
        openssl rand 10485700 2>/dev/null
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c "import os; import sys; sys.stdout.buffer.write(os.urandom(10485700))" 2>/dev/null
    elif command -v python >/dev/null 2>&1; then
        python -c "import os; import sys; sys.stdout.buffer.write(os.urandom(10485700))" 2>/dev/null
    else
        echo "⚠ Using fallback for archive.zip"
        for i in $(seq 1 10); do
            printf '%0999999d' $RANDOM | head -c 1048570
        done
    fi
    
    # ZIP footer
    printf 'PK\x05\x06\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'
} > archive.zip
echo "✓ archive.zip created ($(get_size archive.zip) bytes)"

cd ..

echo ""
echo "=================================================================="
echo "✓ All test files created in: realistic_test_files/"
echo "=================================================================="
echo ""
echo "File Types Created (~10MB each):"
echo ""
echo "HIGHLY COMPRESSIBLE (60-99% compression):"
echo "  ✓ server.log         - Web server logs"
echo "  ✓ api_response.json  - JSON API data"
echo "  ✓ sales_data.csv     - CSV tabular data"
echo "  ✓ config.xml         - XML configuration"
echo "  ✓ article.txt        - Plain text"
echo "  ✓ database.sql       - SQL dump"
echo "  ✓ source_code.js     - JavaScript code"
echo "  ✓ webpage.html       - HTML page"
echo ""
echo "POORLY COMPRESSIBLE (0-5% compression):"
echo "  ✓ binary.bin         - Random binary data"
echo "  ✓ encrypted.dat      - Encrypted data"
echo "  ✓ photo.jpg          - JPEG image"
echo "  ✓ archive.zip        - ZIP archive"
echo ""
echo "Binary files created using:"
if command -v openssl >/dev/null 2>&1; then
    echo "  - openssl (truly random - best quality)"
elif command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1; then
    echo "  - Python os.urandom (truly random - good quality)"
else
    echo "  - Fallback method (may show slight compression)"
fi
echo ""
echo "Next steps:"
echo "  cd realistic_test_files"
echo "  ../benchmark_realistic.sh"
echo "=================================================================="