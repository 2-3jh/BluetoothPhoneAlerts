//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxImageBytes = 5 * 1024 * 1024

type phoneMessage struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	TransferID string `json:"transferId"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
	Index      int    `json:"index"`
	Data       string `json:"data"`
}

type incomingImage struct {
	id        string
	name      string
	size      int64
	hash      string
	nextIndex int
	data      []byte
}

func acceptImageFrame(t **incomingImage, frame phoneMessage, directory string) (string, error) {
	switch frame.Type {
	case "image_start":
		if frame.TransferID == "" || len(frame.TransferID) > 100 || frame.Size <= 0 || frame.Size > maxImageBytes || len(frame.SHA256) != 64 {
			return "", fmt.Errorf("无效的图片传输信息")
		}
		if _, err := hex.DecodeString(frame.SHA256); err != nil {
			return "", fmt.Errorf("无效的图片校验值")
		}
		*t = &incomingImage{id: frame.TransferID, name: frame.Name, size: frame.Size,
			hash: strings.ToLower(frame.SHA256), data: make([]byte, 0, frame.Size)}
		return "", nil
	case "image_chunk":
		if *t == nil || frame.TransferID != (*t).id || frame.Index != (*t).nextIndex {
			return "", fmt.Errorf("图片分块顺序错误")
		}
		part, err := base64.StdEncoding.DecodeString(frame.Data)
		if err != nil || len(part) == 0 || len(part) > 4096 || int64(len((*t).data)+len(part)) > (*t).size {
			return "", fmt.Errorf("无效的图片分块")
		}
		(*t).data = append((*t).data, part...)
		(*t).nextIndex++
		return "", nil
	case "image_end":
		if *t == nil || frame.TransferID != (*t).id || int64(len((*t).data)) != (*t).size {
			return "", fmt.Errorf("图片传输不完整")
		}
		actual := fmt.Sprintf("%x", sha256.Sum256((*t).data))
		if actual != (*t).hash {
			return "", fmt.Errorf("图片校验失败")
		}
		path, err := saveReceivedImage(directory, (*t).name, (*t).data)
		*t = nil
		return path, err
	}
	return "", fmt.Errorf("未知的图片消息")
}

func receivedImagesDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base, _ = os.UserConfigDir()
	}
	return filepath.Join(base, "BluetoothPhoneAlerts", "Received")
}

func safeImageName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "/", "\\"))
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, " .")
	if name == "" || name == "." {
		name = "手机图片"
	}
	if len([]rune(name)) > 100 {
		name = string([]rune(name)[:100])
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".heic", ".heif":
	default:
		name += ".img"
	}
	stem := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" ||
		(len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '1' && stem[3] <= '9') {
		name = "_" + name
	}
	return name
}

func saveReceivedImage(directory, name string, data []byte) (string, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	name = safeImageName(name)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for index := 0; index < 1000; index++ {
		candidate := name
		if index > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, index, ext)
		}
		path := filepath.Join(directory, candidate)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		written, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil || written != len(data) {
			_ = os.Remove(path)
			return "", fmt.Errorf("保存图片失败：%v %v", writeErr, closeErr)
		}
		return path, nil
	}
	return "", fmt.Errorf("同名图片过多，无法保存")
}
