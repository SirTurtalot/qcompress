#!/bin/bash

# Benchmark script for realistic test files

cd realistic_test_files 2>/dev/null || {
    echo "Error: realistic_test_files directory not found"
    echo "Run ./create_realistic_files.sh first"
    exit 1
}

# Check if binary exists
if [ ! -f "../qcompress" ]; then
    echo "Building qcompress..."
    cd ..
    go build -o qcompress qcompressstream_optimized.go
    if [ $? -ne 0 ]; then
        echo "ERROR: Build failed"
        exit 1
    fi
    cd realistic_test_files
fi

BINARY="../qcompress"
KEY=$(openssl rand -hex 32 2>/dev/null || xxd -l 32 -p /dev/urandom | tr -d '\n')

echo "=================================================================="
echo "Realistic File Compression Benchmark"
echo "=================================================================="
echo "Binary: $BINARY"
echo ""

# Results header
printf "%-20s %-12s %-12s %-12s %-12s %-12s %-10s\n" \
    "File" "Original" "Encrypted" "Saved" "Enc(s)" "Dec(s)" "Status"
echo "----------------------------------------------------------------------------------------------------"

# Test all files
for file in *.log *.json *.csv *.xml *.txt *.sql *.js *.html *.bin *.dat *.jpg *.zip; do
    [ ! -f "$file" ] && continue
    
    encrypted_file="${file}.enc"
    decrypted_file="${file}.dec"
    
    # Get original size
    orig_bytes=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
    orig_h=$(numfmt --to=iec-i --suffix=B $orig_bytes 2>/dev/null || \
             awk "BEGIN {printf \"%.1fMB\", $orig_bytes/1048576}")
    
    # Encrypt
    start=$(date +%s.%N)
    $BINARY -mode=encrypt -in="$file" -out="$encrypted_file" -key="$KEY" -level=3 2>/dev/null
    enc_time=$(awk "BEGIN {printf \"%.3f\", $(date +%s.%N) - $start}")
    
    if [ ! -f "$encrypted_file" ]; then
        printf "%-20s %-12s %-12s %-12s %-12s %-12s %-10s\n" \
            "$file" "$orig_h" "FAIL" "-" "-" "-" "✗"
        continue
    fi
    
    # Get encrypted size
    enc_bytes=$(stat -f%z "$encrypted_file" 2>/dev/null || stat -c%s "$encrypted_file" 2>/dev/null)
    enc_h=$(numfmt --to=iec-i --suffix=B $enc_bytes 2>/dev/null || \
            awk "BEGIN {printf \"%.1fMB\", $enc_bytes/1048576}")
    
    # Calculate compression ratio
    saved=$(awk "BEGIN {printf \"%.1f%%\", ($orig_bytes - $enc_bytes) * 100.0 / $orig_bytes}")
    
    # Decrypt
    start=$(date +%s.%N)
    $BINARY -mode=decrypt -in="$encrypted_file" -out="$decrypted_file" -key="$KEY" 2>/dev/null
    dec_time=$(awk "BEGIN {printf \"%.3f\", $(date +%s.%N) - $start}")
    
    # Verify
    if cmp -s "$file" "$decrypted_file" 2>/dev/null; then
        status="✓"
    else
        status="✗"
    fi
    
    printf "%-20s %-12s %-12s %-12s %-12s %-12s %-10s\n" \
        "$file" "$orig_h" "$enc_h" "$saved" "$enc_time" "$dec_time" "$status"
    
    rm -f "$decrypted_file"
done

echo ""
echo "=================================================================="
echo "Summary:"
echo "- These realistic files show TRUE compression performance"
echo "- Zstandard algorithm excels at structured/repetitive data"
echo "- Combined encryption + compression in one pass"
echo "- All files verified for integrity"
echo ""
echo "Encrypted files saved as: *.enc"
echo "Cleanup: rm -f *.enc *.dec"
echo "=================================================================="