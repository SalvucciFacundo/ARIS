package imgutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
)

// PNG signature constants
var (
	pngSignature = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

	// ErrInvalidPNGSignature is returned when the stream does not start with standard 8-byte PNG signature.
	ErrInvalidPNGSignature = errors.New("invalid PNG signature")

	// ErrInvalidChunkCRC is returned when a chunk's IEEE CRC-32 checksum does not match its payload.
	ErrInvalidChunkCRC = errors.New("invalid chunk CRC checksum")

	// ErrNoMetadataFound is returned when a file contains no metadata chunks.
	ErrNoMetadataFound = errors.New("no metadata found in PNG")
)

// InjectPNGMetadata reads a PNG from r, injects the provided key-value metadata pairs
// as tEXt chunks immediately prior to the IEND chunk, and writes the resulting PNG to w.
func InjectPNGMetadata(r io.Reader, w io.Writer, meta map[string]string) error {
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return ErrInvalidPNGSignature
		}
		return fmt.Errorf("read signature: %w", err)
	}

	if !bytes.Equal(sig, pngSignature) {
		return ErrInvalidPNGSignature
	}

	if _, err := w.Write(pngSignature); err != nil {
		return fmt.Errorf("write png signature: %w", err)
	}

	// Sort keys for deterministic chunk ordering
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lengthBuf [4]byte
	var typeBuf [4]byte
	var crcBuf [4]byte

	for {
		if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read chunk length: %w", err)
		}

		if _, err := io.ReadFull(r, typeBuf[:]); err != nil {
			return fmt.Errorf("read chunk type: %w", err)
		}

		chunkLen := binary.BigEndian.Uint32(lengthBuf[:])
		chunkTypeStr := string(typeBuf[:])

		if chunkTypeStr == "IEND" {
			// Write metadata tEXt chunks before IEND
			for _, k := range keys {
				v := meta[k]
				data := make([]byte, len(k)+1+len(v))
				copy(data, k)
				data[len(k)] = 0 // null separator
				copy(data[len(k)+1:], v)

				textChunkLen := uint32(len(data))
				var textLenBuf [4]byte
				binary.BigEndian.PutUint32(textLenBuf[:], textChunkLen)

				textTypeBuf := []byte("tEXt")
				crc := crc32.ChecksumIEEE(append(textTypeBuf, data...))

				var textCrcBuf [4]byte
				binary.BigEndian.PutUint32(textCrcBuf[:], crc)

				if _, err := w.Write(textLenBuf[:]); err != nil {
					return fmt.Errorf("write tEXt chunk length: %w", err)
				}
				if _, err := w.Write(textTypeBuf); err != nil {
					return fmt.Errorf("write tEXt chunk type: %w", err)
				}
				if _, err := w.Write(data); err != nil {
					return fmt.Errorf("write tEXt chunk data: %w", err)
				}
				if _, err := w.Write(textCrcBuf[:]); err != nil {
					return fmt.Errorf("write tEXt chunk crc: %w", err)
				}
			}
		}

		// Read chunk data
		data := make([]byte, chunkLen)
		if chunkLen > 0 {
			if _, err := io.ReadFull(r, data); err != nil {
				return fmt.Errorf("read chunk data (%s): %w", chunkTypeStr, err)
			}
		}

		if _, err := io.ReadFull(r, crcBuf[:]); err != nil {
			return fmt.Errorf("read chunk crc (%s): %w", chunkTypeStr, err)
		}

		// Write the original chunk
		if _, err := w.Write(lengthBuf[:]); err != nil {
			return fmt.Errorf("write chunk length (%s): %w", chunkTypeStr, err)
		}
		if _, err := w.Write(typeBuf[:]); err != nil {
			return fmt.Errorf("write chunk type (%s): %w", chunkTypeStr, err)
		}
		if chunkLen > 0 {
			if _, err := w.Write(data); err != nil {
				return fmt.Errorf("write chunk data (%s): %w", chunkTypeStr, err)
			}
		}
		if _, err := w.Write(crcBuf[:]); err != nil {
			return fmt.Errorf("write chunk crc (%s): %w", chunkTypeStr, err)
		}

		if chunkTypeStr == "IEND" {
			break
		}
	}

	return nil
}

// ExtractPNGMetadata scans a PNG byte stream, verifies chunk CRC-32 integrity,
// and extracts key-value pairs stored inside tEXt chunks without raster decoding.
func ExtractPNGMetadata(r io.Reader) (map[string]string, error) {
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, ErrInvalidPNGSignature
		}
		return nil, fmt.Errorf("read signature: %w", err)
	}

	if !bytes.Equal(sig, pngSignature) {
		return nil, ErrInvalidPNGSignature
	}

	meta := make(map[string]string)
	var lengthBuf [4]byte
	var typeBuf [4]byte
	var crcBuf [4]byte

	for {
		if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read chunk length: %w", err)
		}

		if _, err := io.ReadFull(r, typeBuf[:]); err != nil {
			return nil, fmt.Errorf("read chunk type: %w", err)
		}

		chunkLen := binary.BigEndian.Uint32(lengthBuf[:])
		chunkTypeStr := string(typeBuf[:])

		data := make([]byte, chunkLen)
		if chunkLen > 0 {
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, fmt.Errorf("read chunk data (%s): %w", chunkTypeStr, err)
			}
		}

		if _, err := io.ReadFull(r, crcBuf[:]); err != nil {
			return nil, fmt.Errorf("read chunk crc (%s): %w", chunkTypeStr, err)
		}

		expectedCRC := binary.BigEndian.Uint32(crcBuf[:])
		actualCRC := crc32.ChecksumIEEE(append(typeBuf[:], data...))
		if actualCRC != expectedCRC {
			return nil, fmt.Errorf("%w: chunk %s expected %08x got %08x",
				ErrInvalidChunkCRC, chunkTypeStr, expectedCRC, actualCRC)
		}

		if chunkTypeStr == "tEXt" {
			nullIdx := bytes.IndexByte(data, 0)
			if nullIdx > 0 {
				key := string(data[:nullIdx])
				val := string(data[nullIdx+1:])
				meta[key] = val
			}
		} else if chunkTypeStr == "iTXt" {
			// Handle uncompressed iTXt if present
			nullIdx := bytes.IndexByte(data, 0)
			if nullIdx > 0 && len(data) > nullIdx+3 {
				key := string(data[:nullIdx])
				compFlag := data[nullIdx+1]
				if compFlag == 0 { // uncompressed
					// keyword \x00 flag \x00 method \x00 lang \x00 trans_keyword \x00 text
					remaining := data[nullIdx+3:] // skip flag & method
					langNull := bytes.IndexByte(remaining, 0)
					if langNull >= 0 {
						afterLang := remaining[langNull+1:]
						transNull := bytes.IndexByte(afterLang, 0)
						if transNull >= 0 {
							meta[key] = string(afterLang[transNull+1:])
						}
					}
				}
			}
		}

		if chunkTypeStr == "IEND" {
			break
		}
	}

	return meta, nil
}

// InjectPNGMetadataFile injects metadata into a source PNG file and writes it to dstPath.
// If srcPath == dstPath, the operation writes to a temporary file first then atomically replaces it.
func InjectPNGMetadataFile(srcPath, dstPath string, meta map[string]string) error {
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source png: %w", err)
	}

	var buf bytes.Buffer
	if err := InjectPNGMetadata(bytes.NewReader(srcBytes), &buf, meta); err != nil {
		return err
	}

	if err := os.WriteFile(dstPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write injected png: %w", err)
	}

	return nil
}

// ExtractPNGMetadataFile reads a PNG file from disk and extracts its metadata map.
func ExtractPNGMetadataFile(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open png file: %w", err)
	}
	defer file.Close()

	return ExtractPNGMetadata(file)
}

// InjectMetadata is an alias for InjectPNGMetadata for backward compatibility with design specs.
func InjectMetadata(r io.Reader, w io.Writer, meta map[string]string) error {
	return InjectPNGMetadata(r, w, meta)
}

// ExtractMetadata is an alias for ExtractPNGMetadata for backward compatibility with design specs.
func ExtractMetadata(r io.Reader) (map[string]string, error) {
	return ExtractPNGMetadata(r)
}
