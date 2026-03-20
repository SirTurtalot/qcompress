package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/argon2"
)

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const (
	bufferSize = 1024 * 1024 // 1 MB I/O buffer

	// File format (v1):
	// [magic 4B][flags 1B][salt? 32B][IV 16B][AES-CTR(zstd(plaintext))][HMAC-SHA256 32B]
	// HMAC covers every byte from magic through end of ciphertext.
	fileMagic   = "QCS1"
	flagRawKey  = byte(0x00) // key supplied directly
	flagPassKey = byte(0x01) // key derived from passphrase via Argon2id

	ivLen   = 16          // aes.BlockSize
	macLen  = sha256.Size // 32
	saltLen = 32

	// Argon2id parameters — OWASP interactive-login minimums.
	argon2Time    uint32 = 1
	argon2Memory  uint32 = 64 * 1024 // 64 MB
	argon2Threads uint8  = 4
	argon2KeyLen  uint32 = 32
)

// ─────────────────────────────────────────────────────────────────────────────
// Key helpers
// ─────────────────────────────────────────────────────────────────────────────

// deriveKey produces a 32-byte AES key from a passphrase and salt via Argon2id.
func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
}

// loadRawKey resolves a 32-byte key from flags or the QCOMPRESS_KEY env var.
// Priority: -keyfile > -key flag > QCOMPRESS_KEY env var.
func loadRawKey(keyHex, keyFile string) ([]byte, error) {
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("reading key file: %w", err)
		}
		// Accept 32 raw bytes or a 64-char hex string.
		trimmed := strings.TrimSpace(string(data))
		if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
		if len(data) == 32 {
			return data, nil
		}
		return nil, errors.New("key file must contain 32 raw bytes or a 64-char hex string")
	}

	src := keyHex
	if src == "" {
		src = os.Getenv("QCOMPRESS_KEY")
	}
	if src == "" {
		return nil, errors.New("no key: use -key, -keyfile, -password, or QCOMPRESS_KEY env var")
	}
	key, err := hex.DecodeString(src)
	if err != nil || len(key) != 32 {
		return nil, errors.New("-key / QCOMPRESS_KEY must be exactly 64 hex characters (32 bytes)")
	}
	return key, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Progress bar
// ─────────────────────────────────────────────────────────────────────────────

// newBar returns a progress bar writing to stderr.
// Pass total=-1 for an indeterminate spinner (e.g. unknown stdin size).
func newBar(total int64, desc string) *progressbar.ProgressBar {
	base := []progressbar.Option{
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
	}
	if total < 0 {
		return progressbar.NewOptions(-1, append(base, progressbar.OptionSpinnerType(14))...)
	}
	return progressbar.NewOptions64(total, base...)
}

// ─────────────────────────────────────────────────────────────────────────────
// Encrypt
// ─────────────────────────────────────────────────────────────────────────────

// encrypt compresses then encrypts input, writing to output.
//
// Data flow:  plaintext → zstd → AES-256-CTR → output
//                                            → HMAC (accumulated)
//
// The HMAC is keyed with the AES key and covers:
//   magic + flags + [salt] + IV + ciphertext
//
// The final 32-byte HMAC is appended after the ciphertext.
func encrypt(
	input io.Reader,
	output io.Writer,
	key []byte,
	isPassword bool,
	salt []byte,
	level int,
	inputSize int64,
) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("creating AES cipher: %w", err)
	}

	iv := make([]byte, ivLen)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return fmt.Errorf("generating IV: %w", err)
	}

	// mac mirrors every byte written to output (except the final MAC itself).
	mac := hmac.New(sha256.New, key)
	mw := io.MultiWriter(output, mac)

	// ── Header ──
	write := func(data []byte) error { _, err := mw.Write(data); return err }
	if err := write([]byte(fileMagic)); err != nil {
		return err
	}
	flags := flagRawKey
	if isPassword {
		flags = flagPassKey
	}
	if err := write([]byte{flags}); err != nil {
		return err
	}
	if isPassword {
		if err := write(salt); err != nil {
			return err
		}
	}
	if err := write(iv); err != nil {
		return err
	}

	// ── Compression pipeline → CTR encryption → MultiWriter ──
	stream := cipher.NewCTR(block, iv)
	cw := &cipher.StreamWriter{S: stream, W: mw}

	var zstdLevel zstd.EncoderLevel
	switch level {
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
		zstd.WithWindowSize(1<<20),
	)
	if err != nil {
		return fmt.Errorf("creating zstd encoder: %w", err)
	}

	bar := newBar(inputSize, "Encrypting ")
	buf := make([]byte, bufferSize)
	if _, err := io.CopyBuffer(encoder, io.TeeReader(input, bar), buf); err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("finalizing compression: %w", err)
	}
	bar.Finish()

	// ── Append HMAC (not MAC-covered itself) ──
	if _, err := output.Write(mac.Sum(nil)); err != nil {
		return fmt.Errorf("writing HMAC: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Decrypt
// ─────────────────────────────────────────────────────────────────────────────

// decrypt authenticates, decrypts, and decompresses a QCompress file.
// inputFile must be a seekable *os.File; stdin is not supported for decryption.
func decrypt(
	inputFile *os.File,
	output io.Writer,
	keyHex, keyFile, password string,
) error {
	// ── Magic ──
	magicBuf := make([]byte, len(fileMagic))
	if _, err := io.ReadFull(inputFile, magicBuf); err != nil {
		return errors.New("failed to read file header")
	}
	if string(magicBuf) != fileMagic {
		return errors.New("not a QCompress file (invalid magic) — was this encrypted with qcompress?")
	}

	// ── Flags ──
	flagBuf := make([]byte, 1)
	if _, err := io.ReadFull(inputFile, flagBuf); err != nil {
		return errors.New("failed to read format flags")
	}
	flags := flagBuf[0]

	// ── Resolve key ──
	var (
		key     []byte
		saltBuf []byte
	)
	switch flags {
	case flagRawKey:
		if password != "" {
			return errors.New("file was encrypted with a raw key; use -key / -keyfile / QCOMPRESS_KEY, not -password")
		}
		var err error
		key, err = loadRawKey(keyHex, keyFile)
		if err != nil {
			return err
		}

	case flagPassKey:
		if password == "" {
			return errors.New("file was encrypted with a passphrase; provide -password to decrypt")
		}
		saltBuf = make([]byte, saltLen)
		if _, err := io.ReadFull(inputFile, saltBuf); err != nil {
			return errors.New("failed to read Argon2 salt from header")
		}
		t0 := time.Now()
		fmt.Fprint(os.Stderr, "Deriving key from password... ")
		key = deriveKey(password, saltBuf)
		fmt.Fprintf(os.Stderr, "done (%s)\n", time.Since(t0).Round(time.Millisecond))

	default:
		return fmt.Errorf("unsupported format flags 0x%02x — file may require a newer version of qcompress", flags)
	}

	// ── IV ──
	iv := make([]byte, ivLen)
	if _, err := io.ReadFull(inputFile, iv); err != nil {
		return errors.New("failed to read IV")
	}

	// ── Ciphertext boundaries ──
	ciphertextStart, err := inputFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("seeking: %w", err)
	}
	fi, err := inputFile.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	ciphertextLen := fi.Size() - ciphertextStart - int64(macLen)
	if ciphertextLen < 0 {
		return errors.New("invalid file: too short to contain a valid HMAC")
	}

	// ── Verify HMAC before decrypting (authenticate-then-decrypt) ──
	// Re-feed the header bytes already consumed from the reader.
	mac := hmac.New(sha256.New, key)
	mac.Write(magicBuf)
	mac.Write(flagBuf)
	if saltBuf != nil {
		mac.Write(saltBuf)
	}
	mac.Write(iv)

	buf := make([]byte, bufferSize)
	if _, err := io.CopyBuffer(mac, io.LimitReader(inputFile, ciphertextLen), buf); err != nil {
		return fmt.Errorf("reading ciphertext for MAC verification: %w", err)
	}
	storedMAC := make([]byte, macLen)
	if _, err := io.ReadFull(inputFile, storedMAC); err != nil {
		return errors.New("failed to read stored HMAC")
	}
	// Constant-time comparison prevents timing attacks.
	if !hmac.Equal(mac.Sum(nil), storedMAC) {
		return errors.New("integrity check failed — wrong key/password, or file is corrupted/tampered")
	}

	// ── Seek back to start of ciphertext ──
	if _, err := inputFile.Seek(ciphertextStart, io.SeekStart); err != nil {
		return fmt.Errorf("seeking to ciphertext start: %w", err)
	}

	// ── Decrypt + Decompress ──
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("creating AES cipher: %w", err)
	}
	stream := cipher.NewCTR(block, iv)
	limitedR := io.LimitReader(inputFile, ciphertextLen)

	// Progress bar tracks ciphertext bytes consumed (known size).
	bar := newBar(ciphertextLen, "Decrypting ")
	cipherR := &cipher.StreamReader{S: stream, R: io.TeeReader(limitedR, bar)}

	decoder, err := zstd.NewReader(cipherR,
		zstd.WithDecoderConcurrency(runtime.GOMAXPROCS(0)),
	)
	if err != nil {
		return fmt.Errorf("creating zstd decoder: %w", err)
	}
	defer decoder.Close() // releases goroutines; zstd.Decoder.Close() has no return value

	if _, err := io.CopyBuffer(output, decoder, buf); err != nil {
		return fmt.Errorf("decrypting/decompressing: %w", err)
	}
	bar.Finish()
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mode     := flag.String("mode", "", "encrypt or decrypt")
	inPath   := flag.String("in", "", "input file (use - for stdin; decrypt requires a real file)")
	outPath  := flag.String("out", "", "output file (use - for stdout; default: stdout)")
	keyHex   := flag.String("key", "", "32-byte key as 64 hex chars (or set QCOMPRESS_KEY env var)")
	keyFile  := flag.String("keyfile", "", "path to key file (32 raw bytes or 64-char hex string)")
	password := flag.String("password", "", "passphrase — Argon2id key derivation (encrypt: auto-generates salt; decrypt: reads salt from file)")
	level    := flag.Int("level", 3, "zstd compression level 1-4 (encrypt only; default 3)")
	flag.Parse()

	if *mode == "" {
		fmt.Fprintln(os.Stderr, "Usage: qcompress -mode=encrypt|decrypt -in=<file> -out=<file> [key option] [-level=1-4]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Key options (choose exactly one):")
		fmt.Fprintln(os.Stderr, "  -key=<64 hex chars>      Raw 32-byte key as hex")
		fmt.Fprintln(os.Stderr, "  -keyfile=<path>          File containing key (32 raw bytes or hex)")
		fmt.Fprintln(os.Stderr, "  -password=<passphrase>   Passphrase (Argon2id key derivation)")
		fmt.Fprintln(os.Stderr, "  QCOMPRESS_KEY=<hex>      Env var equivalent of -key (avoids shell history)")
		os.Exit(1)
	}

	// Exactly one key source must be provided.
	n := 0
	if *keyHex != ""   { n++ }
	if *keyFile != ""  { n++ }
	if *password != "" { n++ }
	if n == 0 && os.Getenv("QCOMPRESS_KEY") != "" { n++ }
	if n == 0 {
		log.Fatal("No key provided: use -key, -keyfile, -password, or QCOMPRESS_KEY env var")
	}
	if n > 1 {
		log.Fatal("Provide exactly one of: -key, -keyfile, -password, or QCOMPRESS_KEY env var")
	}

	// ── Open output ──
	var (
		outFile *os.File
		output  io.Writer
	)
	if *outPath == "" || *outPath == "-" {
		output = os.Stdout
	} else {
		f, err := os.Create(*outPath)
		if err != nil {
			log.Fatal(err)
		}
		outFile = f
		output = f
	}

	// die cleans up a partial output file before calling log.Fatal.
	// Always use die() (never log.Fatal) after the output file is open.
	die := func(args ...any) {
		if outFile != nil {
			outFile.Close()
			os.Remove(*outPath)
		}
		log.Fatal(args...)
	}

	// ── Dispatch ──
	var opErr error

	switch *mode {
	case "encrypt":
		// Open input.
		var (
			input     io.Reader
			inputSize int64 = -1 // -1 = unknown (stdin → spinner progress)
		)
		if *inPath == "" || *inPath == "-" {
			input = os.Stdin
		} else {
			f, err := os.Open(*inPath)
			if err != nil {
				die(err)
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				die(err)
			}
			inputSize = fi.Size()
			input = f
		}

		// Resolve key.
		var (
			key        []byte
			isPassword bool
			salt       []byte
		)
		if *password != "" {
			isPassword = true
			salt = make([]byte, saltLen)
			if _, err := io.ReadFull(rand.Reader, salt); err != nil {
				die("generating Argon2 salt:", err)
			}
			t0 := time.Now()
			fmt.Fprint(os.Stderr, "Deriving key from password... ")
			key = deriveKey(*password, salt)
			fmt.Fprintf(os.Stderr, "done (%s)\n", time.Since(t0).Round(time.Millisecond))
		} else {
			var err error
			key, err = loadRawKey(*keyHex, *keyFile)
			if err != nil {
				die(err)
			}
		}

		opErr = encrypt(input, output, key, isPassword, salt, *level, inputSize)

	case "decrypt":
		if *inPath == "" || *inPath == "-" {
			die("decrypt requires a real file path for -in (stdin is not supported for decryption)")
		}
		inFile, err := os.Open(*inPath)
		if err != nil {
			die(err)
		}
		defer inFile.Close()

		opErr = decrypt(inFile, output, *keyHex, *keyFile, *password)

	default:
		die(fmt.Sprintf("invalid mode %q — use encrypt or decrypt", *mode))
	}

	if opErr != nil {
		die(opErr)
	}

	// ── Flush to disk before reporting success ──
	if outFile != nil {
		if err := outFile.Sync(); err != nil {
			die("fsync:", err)
		}
		if err := outFile.Close(); err != nil {
			die("closing output:", err)
		}
	}
}
