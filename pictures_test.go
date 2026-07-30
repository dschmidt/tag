// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"os"
	"testing"
)

func readPictures(t *testing.T, path string) []Picture {
	t.Helper()
	f, err := os.Open("testdata/" + path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	m, err := ReadFrom(f)
	if err != nil {
		t.Fatalf("ReadFrom %s: %v", path, err)
	}
	return m.Pictures()
}

func TestPictures(t *testing.T) {
	// sample.multipage.ogg carries a single front cover.
	pics := readPictures(t, "with_tags/sample.multipage.ogg")
	if len(pics) != 1 {
		t.Fatalf("expected 1 picture, got %d", len(pics))
	}
	if got := pics[0].RawType; got != 0x03 {
		t.Errorf("RawType = 0x%02x, want 0x03 (front cover)", got)
	}
	if got := pics[0].Type; got != "Cover (front)" {
		t.Errorf("Type = %q, want %q", got, "Cover (front)")
	}
	if len(pics[0].Data) == 0 {
		t.Error("picture data is empty")
	}
}

func TestPicturesNone(t *testing.T) {
	// Samples without embedded artwork must return no pictures.
	if pics := readPictures(t, "with_tags/sample.id3v24.mp3"); pics != nil {
		t.Errorf("expected nil, got %d pictures", len(pics))
	}
}
