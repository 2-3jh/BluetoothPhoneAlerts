//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestImageTransferSavesVerifiedImage(t *testing.T) {
	data := []byte("sample image bytes")
	var transfer *incomingImage
	directory := t.TempDir()
	frames := []phoneMessage{
		{Type: "image_start", TransferID: "test-transfer", Name: `..\photo.png`, Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))},
		{Type: "image_chunk", TransferID: "test-transfer", Index: 0, Data: base64.StdEncoding.EncodeToString(data)},
		{Type: "image_end", TransferID: "test-transfer"},
	}
	for index, frame := range frames {
		path, err := acceptImageFrame(&transfer, frame, directory)
		if err != nil {
			t.Fatalf("frame %d: %v", index, err)
		}
		if index == len(frames)-1 {
			if filepath.Dir(path) != directory || filepath.Base(path) != "photo.png" {
				t.Fatalf("unsafe output path: %s", path)
			}
			saved, err := os.ReadFile(path)
			if err != nil || string(saved) != string(data) {
				t.Fatalf("saved image mismatch: %q, %v", saved, err)
			}
		}
	}
}

func TestImageTransferRejectsBadOrderAndHash(t *testing.T) {
	data := []byte("image")
	var transfer *incomingImage
	start := phoneMessage{Type: "image_start", TransferID: "id", Name: "photo.jpg", Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	if _, err := acceptImageFrame(&transfer, start, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := acceptImageFrame(&transfer, phoneMessage{Type: "image_chunk", TransferID: "id", Index: 1, Data: base64.StdEncoding.EncodeToString(data)}, t.TempDir()); err == nil {
		t.Fatal("out-of-order image chunk accepted")
	}
	if _, err := acceptImageFrame(&transfer, phoneMessage{Type: "image_chunk", TransferID: "id", Index: 0, Data: base64.StdEncoding.EncodeToString([]byte("wrong"))}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := acceptImageFrame(&transfer, phoneMessage{Type: "image_end", TransferID: "id"}, t.TempDir()); err == nil {
		t.Fatal("corrupt image accepted")
	}
}

func TestImageTransferRejectsOversize(t *testing.T) {
	var transfer *incomingImage
	_, err := acceptImageFrame(&transfer, phoneMessage{Type: "image_start", TransferID: "id", Size: maxImageBytes + 1, SHA256: fmt.Sprintf("%064d", 0)}, t.TempDir())
	if err == nil {
		t.Fatal("oversize image accepted")
	}
}
