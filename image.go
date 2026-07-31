// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import "bytes"

// imageExtFromMIME maps a known image MIME type to a file extension. It returns
// an empty string for MIME types it does not recognise.
func imageExtFromMIME(mime string) string {
	switch mime {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/bmp", "image/x-ms-bmp":
		return "bmp"
	case "image/tiff":
		return "tif"
	case "image/webp":
		return "webp"
	}
	return ""
}

// sniffImageType inspects the leading bytes of an image and returns its MIME type
// and file extension based on the file's magic number. It returns two empty
// strings when the format is not recognised.
func sniffImageType(b []byte) (mime, ext string) {
	switch {
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return "image/jpeg", "jpg"
	case bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png", "png"
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return "image/gif", "gif"
	case bytes.HasPrefix(b, []byte("BM")):
		return "image/bmp", "bmp"
	case bytes.HasPrefix(b, []byte{'I', 'I', 0x2A, 0x00}), bytes.HasPrefix(b, []byte{'M', 'M', 0x00, 0x2A}):
		return "image/tiff", "tif"
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "image/webp", "webp"
	}
	return "", ""
}

// resolveImageType returns the MIME type and extension for picture data.
// Byte sniffing wins so missing or wrong declared types are corrected; the
// declared MIME type is only a fallback when the bytes aren't recognised.
func resolveImageType(declaredMIME string, data []byte) (mime, ext string) {
	if m, e := sniffImageType(data); m != "" {
		return m, e
	}
	return declaredMIME, imageExtFromMIME(declaredMIME)
}
