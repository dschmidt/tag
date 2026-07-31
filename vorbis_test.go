// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

func buildPictureBlock(picType uint32, mime string, data []byte) []byte {
	buf := &bytes.Buffer{}
	be := func(n uint32) { binary.Write(buf, binary.BigEndian, n) }
	be(picType)
	be(uint32(len(mime)))
	buf.WriteString(mime)
	be(0)                 // description length
	be(0)                 // width
	be(0)                 // height
	be(0)                 // color depth
	be(0)                 // colors used
	be(uint32(len(data))) // data length
	buf.Write(data)
	return buf.Bytes()
}

func buildVorbisComment(comments ...string) []byte {
	buf := &bytes.Buffer{}
	le := func(n uint32) { binary.Write(buf, binary.LittleEndian, n) }
	const vendor = "test"
	le(uint32(len(vendor)))
	buf.WriteString(vendor)
	le(uint32(len(comments)))
	for _, c := range comments {
		le(uint32(len(c)))
		buf.WriteString(c)
	}
	return buf.Bytes()
}

// A malformed embedded picture (here an out-of-range picture type) must not sink
// the rest of the tags: title/artist stay readable and the bad picture is dropped.
func TestVorbisMalformedPictureKeepsTags(t *testing.T) {
	bad := base64.StdEncoding.EncodeToString(buildPictureBlock(0xFF, "image/png", []byte{0x89, 'P', 'N', 'G'}))
	comment := buildVorbisComment("TITLE=Hello", "METADATA_BLOCK_PICTURE="+bad, "ARTIST=World")

	m := newMetadataVorbis()
	if err := m.readVorbisComment(bytes.NewReader(comment)); err != nil {
		t.Fatalf("readVorbisComment: unexpected error: %v", err)
	}
	if got := m.Title(); got != "Hello" {
		t.Errorf("Title = %q, want %q", got, "Hello")
	}
	if got := m.Artist(); got != "World" {
		t.Errorf("Artist = %q, want %q", got, "World")
	}
	if len(m.ps) != 0 {
		t.Errorf("expected the malformed picture to be skipped, got %d", len(m.ps))
	}
}

// A valid embedded picture is still decoded.
func TestVorbisValidPictureDecoded(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	good := base64.StdEncoding.EncodeToString(buildPictureBlock(0x03, "image/png", png))
	comment := buildVorbisComment("TITLE=Hello", "METADATA_BLOCK_PICTURE="+good)

	m := newMetadataVorbis()
	if err := m.readVorbisComment(bytes.NewReader(comment)); err != nil {
		t.Fatalf("readVorbisComment: unexpected error: %v", err)
	}
	if len(m.ps) != 1 {
		t.Fatalf("expected 1 picture, got %d", len(m.ps))
	}
	if got := m.ps[0].MIMEType; got != "image/png" {
		t.Errorf("MIMEType = %q, want image/png", got)
	}
}
