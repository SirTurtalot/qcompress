package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/klauspost/compress/zstd"
)

const (
	bufferSize = 1024 * 1024 // 1MB buffer for better I/O performance
)

func main() {
	m := flag.String("mode", "", "encrypt or decrypt")
	i := flag.String("in", "", "input file")
	o := flag.String("out", "", "output file")
	k := flag.String("key", "", "32-byte hex key")
	level := flag.Int("level", 3, "compression level (1-4: fastest-best, default 3)")
	flag.Parse()

	if *m == "" || *i == "" || *o == "" || *k == "" {
		log.Fatal("Usage: -mode=encrypt/decrypt -in=input -out=output -key=64hexdigits [-level=1-4]")
	}

	key, err := hex.DecodeString(*k)
	if err != nil || len(key) != 32 {
		log.Fatal("Invalid 32-byte hex key")
	}

	input, err := os.Open(*i)
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()

	output, err := os.Create(*o)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatal(err)
	}

	iv := make([]byte, aes.BlockSize)

	switch *m {
	case "encrypt":
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			log.Fatal(err)
		}
		
		mac := hmac.New(sha256.New, key)
		mw := io.MultiWriter(output, mac)
		
		if _, err := mw.Write(iv); err != nil {
			log.Fatal(err)
		}
		
		stream := cipher.NewCTR(block, iv)
		cw := &cipher.StreamWriter{S: stream, W: mw}
		
		// Use zstd instead of gzip - much faster with similar/better compression
		var zstdLevel zstd.EncoderLevel
		switch *level {
		case 1:
			zstdLevel = zstd.SpeedFastest
		case 2:
			zstdLevel = zstd.SpeedDefault
		case 3:
			zstdLevel = zstd.SpeedBetterCompression
		case 4:
			zstdLevel = zstd.SpeedBestCompression
		default:
			zstdLevel = zstd.SpeedBetterCompression
		}
		
		encoder, err := zstd.NewWriter(cw,
			zstd.WithEncoderLevel(zstdLevel),
			zstd.WithEncoderConcurrency(runtime.GOMAXPROCS(0)),
			zstd.WithWindowSize(1<<20), // 1MB window
		)
		if err != nil {
			log.Fatal(err)
		}
		
		// Use buffered copy for better performance
		buf := make([]byte, bufferSize)
		if _, err := io.CopyBuffer(encoder, input, buf); err != nil {
			log.Fatal(err)
		}
		
		if err := encoder.Close(); err != nil {
			log.Fatal(err)
		}
		
		if _, err := output.Write(mac.Sum(nil)); err != nil {
			log.Fatal(err)
		}
		
	case "decrypt":
		if _, err := io.ReadFull(input, iv); err != nil {
			log.Fatal(err)
		}
		
		fi, err := input.Stat()
		if err != nil {
			log.Fatal(err)
		}
		
		size := fi.Size()
		ivSize := int64(len(iv))
		macSize := int64(sha256.Size)
		
		if size < ivSize+macSize {
			log.Fatal("Invalid file: too short")
		}
		
		// Read MAC
		if _, err := input.Seek(size-macSize, io.SeekStart); err != nil {
			log.Fatal(err)
		}
		receivedMac := make([]byte, macSize)
		if _, err := io.ReadFull(input, receivedMac); err != nil {
			log.Fatal(err)
		}
		
		// Verify MAC
		if _, err := input.Seek(ivSize, io.SeekStart); err != nil {
			log.Fatal(err)
		}
		mac := hmac.New(sha256.New, key)
		mac.Write(iv)
		ciphertextLen := size - ivSize - macSize
		
		// Use buffered reading for MAC verification
		buf := make([]byte, bufferSize)
		if _, err := io.CopyBuffer(mac, io.LimitReader(input, ciphertextLen), buf); err != nil {
			log.Fatal(err)
		}
		
		if !hmac.Equal(mac.Sum(nil), receivedMac) {
			log.Fatal("Integrity check failed")
		}
		
		// Decrypt
		if _, err := input.Seek(ivSize, io.SeekStart); err != nil {
			log.Fatal(err)
		}
		
		stream := cipher.NewCTR(block, iv)
		limitedR := io.LimitReader(input, ciphertextLen)
		cipherR := cipher.StreamReader{S: stream, R: limitedR}
		
		decoder, err := zstd.NewReader(&cipherR,
			zstd.WithDecoderConcurrency(runtime.GOMAXPROCS(0)),
		)
		if err != nil {
			log.Fatal(err)
		}
		defer decoder.Close()
		
		// Use buffered copy for decompression
		if _, err := io.CopyBuffer(output, decoder, buf); err != nil {
			log.Fatal(err)
		}
		
	default:
		log.Fatal("Invalid mode")
	}
}