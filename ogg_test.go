// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"os"
	"testing"
)

// TestOGGCodecs verifies that tag reading works across the Ogg-mapped codecs,
// including the ones beyond plain Vorbis/Opus: Speex, Theora and FLAC-in-Ogg.
func TestOGGCodecs(t *testing.T) {
	files := []string{
		"with_tags/testFLAC.oga",   // FLAC-in-Ogg
		"with_tags/testSPEEX.spx",  // Speex
		"with_tags/testTHEORA.ogv", // Theora
		"with_tags/testOPUS.opus",  // Opus
		"with_tags/sample.ogg",     // Vorbis
	}

	for _, f := range files {
		fh, err := os.Open("testdata/" + f)
		if err != nil {
			t.Fatalf("open %s: %v", f, err)
		}
		m, err := ReadFrom(fh)
		fh.Close()
		if err != nil {
			t.Errorf("%s: ReadFrom: %v", f, err)
			continue
		}
		if m.FileType() != OGG {
			t.Errorf("%s: FileType = %q, want OGG", f, m.FileType())
		}
		if m.Title() == "" {
			t.Errorf("%s: expected a title, got empty", f)
		}
	}
}
