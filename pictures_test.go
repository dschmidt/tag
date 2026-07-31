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
	if got := pics[0].MIMEType; got != "image/png" {
		t.Errorf("MIMEType = %q, want %q", got, "image/png")
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

// TestMultiplePictures covers files with more than one embedded cover across
// the container formats (Tika TIKA-4801 fixtures). The samples carry a front
// and a back cover.
func TestMultiplePictures(t *testing.T) {
	files := []string{
		"with_tags/testVORBIS_twoCovers.ogg",
		"with_tags/testFLAC_twoCovers.flac",
		"with_tags/testMP3v23_twoCovers.mp3",
		"with_tags/testMP4_twoCovers.m4a",
	}

	for _, f := range files {
		pics := readPictures(t, f)
		if len(pics) != 2 {
			t.Errorf("%s: expected 2 pictures, got %d", f, len(pics))
			continue
		}
		for i, p := range pics {
			if len(p.Data) == 0 {
				t.Errorf("%s [%d]: picture data is empty", f, i)
			}
			if p.MIMEType == "" {
				t.Errorf("%s [%d]: MIME type is empty", f, i)
			}
		}
	}
}

// TestPictureMIMESniffed verifies that the embedded image format is resolved
// from the picture bytes, even for MP4 where the atom carries only a numeric
// type code.
func TestPictureMIMESniffed(t *testing.T) {
	pics := readPictures(t, "with_tags/testMP4_twoCovers.m4a")
	if len(pics) != 2 {
		t.Fatalf("expected 2 pictures, got %d", len(pics))
	}
	got := map[string]bool{}
	for _, p := range pics {
		got[p.MIMEType] = true
	}
	for _, want := range []string{"image/png", "image/jpeg"} {
		if !got[want] {
			t.Errorf("expected a %s cover, got %v", want, got)
		}
	}
}
