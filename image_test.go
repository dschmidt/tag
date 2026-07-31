// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import "testing"

func TestSniffImageType(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		mime string
		ext  string
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg", "jpg"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "image/png", "png"},
		{"gif87", []byte("GIF87a...."), "image/gif", "gif"},
		{"gif89", []byte("GIF89a...."), "image/gif", "gif"},
		{"bmp", []byte("BMxxxx"), "image/bmp", "bmp"},
		{"tiff-le", []byte{'I', 'I', 0x2A, 0x00}, "image/tiff", "tif"},
		{"tiff-be", []byte{'M', 'M', 0x00, 0x2A}, "image/tiff", "tif"},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp", "webp"},
		{"unknown", []byte("not an image"), "", ""},
		{"empty", nil, "", ""},
	}

	for _, tt := range tests {
		mime, ext := sniffImageType(tt.data)
		if mime != tt.mime || ext != tt.ext {
			t.Errorf("%s: sniffImageType = (%q, %q), want (%q, %q)", tt.name, mime, ext, tt.mime, tt.ext)
		}
	}
}

func TestResolveImageType(t *testing.T) {
	gif := []byte("GIF89a....")
	tests := []struct {
		name     string
		declared string
		data     []byte
		mime     string
		ext      string
	}{
		{"sniff over missing", "", gif, "image/gif", "gif"},
		{"sniff over wrong", "image/jpeg", gif, "image/gif", "gif"},
		{"declared fallback", "image/png", []byte("no magic"), "image/png", "png"},
		{"unknown declared", "application/octet-stream", []byte("no magic"), "application/octet-stream", ""},
	}

	for _, tt := range tests {
		mime, ext := resolveImageType(tt.declared, tt.data)
		if mime != tt.mime || ext != tt.ext {
			t.Errorf("%s: resolveImageType = (%q, %q), want (%q, %q)", tt.name, mime, ext, tt.mime, tt.ext)
		}
	}
}
