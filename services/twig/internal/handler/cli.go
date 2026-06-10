package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type CLIBinary struct {
	available bool
	path      string
	version   string
	filename  string
	os        string
	arch      string
	label     string
	size      int64
	sha256    string
}

func NewCLIBinary(dir string) *CLIBinary {
	c := &CLIBinary{
		filename: "twig",
		os:       "linux",
		arch:     "amd64",
		label:    "Linux (x86-64)",
		version:  "dev",
	}

	binaryPath := filepath.Join(dir, "twig")
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return c
	}

	info, err := os.Stat(binaryPath)
	if err != nil {
		return c
	}

	sum := sha256.Sum256(data)
	c.path = binaryPath
	c.size = info.Size()
	c.sha256 = hex.EncodeToString(sum[:])
	c.available = true

	if raw, err := os.ReadFile(filepath.Join(dir, "version")); err == nil {
		if v := strings.TrimSpace(string(raw)); v != "" {
			c.version = v
		}
	}

	return c
}

func (c *CLIBinary) ServeDownload(w http.ResponseWriter, r *http.Request) {
	if !c.available {
		http.Error(w, "binary unavailable", http.StatusServiceUnavailable)
		return
	}
	f, err := os.Open(c.path)
	if err != nil {
		http.Error(w, "binary unavailable", http.StatusServiceUnavailable)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="twig"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", c.size))
	io.Copy(w, f)
}

func (c *CLIBinary) ServeInfo(w http.ResponseWriter, r *http.Request) {
	if !c.available {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "binary unavailable"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"version":  c.version,
		"filename": c.filename,
		"os":       c.os,
		"arch":     c.arch,
		"label":    c.label,
		"size":     c.size,
		"sha256":   c.sha256,
	})
}
