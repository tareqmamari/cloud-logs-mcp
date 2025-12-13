// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements response compression for reducing bandwidth usage.
package tools

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"sync"
)

// CompressionLevel defines compression aggressiveness
type CompressionLevel int

// Compression levels
const (
	CompressionNone    CompressionLevel = 0
	CompressionFast    CompressionLevel = 1
	CompressionDefault CompressionLevel = 6
	CompressionBest    CompressionLevel = 9
)

// CompressionStats tracks compression efficiency
type CompressionStats struct {
	OriginalSize   int     `json:"original_size"`
	CompressedSize int     `json:"compressed_size"`
	Ratio          float64 `json:"compression_ratio"`
	Algorithm      string  `json:"algorithm"`
}

// Buffer pool for compression
var compressionBufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// CompressJSON compresses JSON data using gzip.
// Returns the compressed data and compression statistics.
func CompressJSON(data interface{}) ([]byte, *CompressionStats, error) {
	// Marshal to JSON first
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}

	originalSize := len(jsonData)

	// Don't compress small payloads (overhead not worth it)
	if originalSize < 1024 {
		return jsonData, &CompressionStats{
			OriginalSize:   originalSize,
			CompressedSize: originalSize,
			Ratio:          1.0,
			Algorithm:      "none",
		}, nil
	}

	// Get buffer from pool
	buf := compressionBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer compressionBufferPool.Put(buf)

	// Create gzip writer
	gzWriter, err := gzip.NewWriterLevel(buf, int(CompressionDefault))
	if err != nil {
		return nil, nil, err
	}

	// Write compressed data
	_, err = gzWriter.Write(jsonData)
	if err != nil {
		return nil, nil, err
	}

	// Close to flush
	if err := gzWriter.Close(); err != nil {
		return nil, nil, err
	}

	compressed := buf.Bytes()
	compressedSize := len(compressed)

	// If compression didn't help, return original
	if compressedSize >= originalSize {
		return jsonData, &CompressionStats{
			OriginalSize:   originalSize,
			CompressedSize: originalSize,
			Ratio:          1.0,
			Algorithm:      "none",
		}, nil
	}

	// Return compressed data
	result := make([]byte, compressedSize)
	copy(result, compressed)

	return result, &CompressionStats{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		Ratio:          float64(originalSize) / float64(compressedSize),
		Algorithm:      "gzip",
	}, nil
}

// DecompressJSON decompresses gzip-compressed JSON data.
func DecompressJSON(data []byte, target interface{}) error {
	// Try to decompress
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		// Not compressed, try direct JSON unmarshal
		return json.Unmarshal(data, target)
	}
	defer func() { _ = gzReader.Close() }()

	// Read decompressed data
	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return err
	}

	return json.Unmarshal(decompressed, target)
}

// ResponseCompressor handles response compression with content awareness
type ResponseCompressor struct {
	minSize int              // Minimum size to compress
	level   CompressionLevel // Compression level
	enabled bool
}

// NewResponseCompressor creates a new response compressor
func NewResponseCompressor(minSize int, level CompressionLevel, enabled bool) *ResponseCompressor {
	if minSize <= 0 {
		minSize = 1024 // Default: 1KB minimum
	}
	return &ResponseCompressor{
		minSize: minSize,
		level:   level,
		enabled: enabled,
	}
}

// CompressResponse compresses a response map if beneficial
func (c *ResponseCompressor) CompressResponse(response map[string]interface{}) ([]byte, *CompressionStats, error) {
	if !c.enabled {
		data, err := json.Marshal(response)
		if err != nil {
			return nil, nil, err
		}
		return data, &CompressionStats{
			OriginalSize:   len(data),
			CompressedSize: len(data),
			Ratio:          1.0,
			Algorithm:      "none",
		}, nil
	}

	return CompressJSON(response)
}

// StreamingCompressor provides streaming compression for large responses
type StreamingCompressor struct {
	buf      *bytes.Buffer
	gzWriter *gzip.Writer
	encoder  *json.Encoder
}

// NewStreamingCompressor creates a new streaming compressor
func NewStreamingCompressor() (*StreamingCompressor, error) {
	buf := new(bytes.Buffer)
	gzWriter, err := gzip.NewWriterLevel(buf, int(CompressionDefault))
	if err != nil {
		return nil, err
	}

	return &StreamingCompressor{
		buf:      buf,
		gzWriter: gzWriter,
		encoder:  json.NewEncoder(gzWriter),
	}, nil
}

// WriteItem writes a single item to the compressed stream
func (s *StreamingCompressor) WriteItem(item interface{}) error {
	return s.encoder.Encode(item)
}

// Finish completes the compression and returns the compressed data
func (s *StreamingCompressor) Finish() ([]byte, error) {
	if err := s.gzWriter.Close(); err != nil {
		return nil, err
	}
	return s.buf.Bytes(), nil
}

// ChunkedResponse represents a response that can be sent in chunks
type ChunkedResponse struct {
	TotalItems  int           `json:"total_items"`
	ChunkNumber int           `json:"chunk_number"`
	TotalChunks int           `json:"total_chunks"`
	Items       []interface{} `json:"items"`
	HasMore     bool          `json:"has_more"`
	NextCursor  string        `json:"next_cursor,omitempty"`
	Compression string        `json:"compression,omitempty"`
}

// ChunkResponse splits a large response into smaller chunks
func ChunkResponse(items []interface{}, chunkSize int) []*ChunkedResponse {
	if chunkSize <= 0 {
		chunkSize = 100 // Default chunk size
	}

	totalItems := len(items)
	totalChunks := (totalItems + chunkSize - 1) / chunkSize
	chunks := make([]*ChunkedResponse, 0, totalChunks)

	for i := 0; i < totalItems; i += chunkSize {
		end := i + chunkSize
		if end > totalItems {
			end = totalItems
		}

		chunkNumber := i/chunkSize + 1
		chunks = append(chunks, &ChunkedResponse{
			TotalItems:  totalItems,
			ChunkNumber: chunkNumber,
			TotalChunks: totalChunks,
			Items:       items[i:end],
			HasMore:     chunkNumber < totalChunks,
		})
	}

	return chunks
}
