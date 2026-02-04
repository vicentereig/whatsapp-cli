// Package types provides shared data structures used across packages.
// This enables dependency inversion: both client and commands import types,
// rather than commands importing client.
package types

// MediaDownloadRequest contains parameters for downloading media from WhatsApp.
// Used by both client (to perform download) and commands (to request download).
type MediaDownloadRequest struct {
	URL           string
	DirectPath    string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    uint64
	MediaType     string
	MimeType      string
}
