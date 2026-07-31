// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var atomTypes = map[int]string{
	0:  "implicit", // automatic based on atom name
	1:  "text",
	13: "jpeg",
	14: "png",
	21: "uint8",
}

// NB: atoms does not include "----", this is handled separately
var atoms = atomNames(map[string]string{
	"\xa9alb": "album",
	"\xa9art": "artist",
	"\xa9ART": "artist",
	"aART":    "album_artist",
	"\xa9day": "year",
	"\xa9nam": "title",
	"\xa9gen": "genre",
	"trkn":    "track",
	"\xa9wrt": "composer",
	"\xa9too": "encoder",
	"cprt":    "copyright",
	"covr":    "picture",
	"\xa9grp": "grouping",
	"keyw":    "keyword",
	"\xa9lyr": "lyrics",
	"\xa9cmt": "comment",
	"tmpo":    "tempo",
	"cpil":    "compilation",
	"disk":    "disc",
})

var means = map[string]bool{
	"com.apple.iTunes":          true,
	"com.mixedinkey.mixedinkey": true,
	"com.serato.dj":             true,
}

type atomNames map[string]string

func (f atomNames) Name(n string) []string {
	res := make([]string, 1)
	for k, v := range f {
		if v == n {
			res = append(res, k)
		}
	}
	return res
}

// metadataMP4 is the implementation of Metadata for MP4 tag (atom) data.
type metadataMP4 struct {
	fileType FileType
	data     map[string]interface{}
}

// ReadAtoms reads MP4 metadata atoms from the io.ReadSeeker into a Metadata, returning
// non-nil error if there was a problem.
func ReadAtoms(r io.ReadSeeker) (Metadata, error) {
	m := metadataMP4{
		data:     make(map[string]interface{}),
		fileType: UnknownFileType,
	}
	err := m.readAtoms(r)
	return m, err
}

func (m metadataMP4) readAtoms(r io.ReadSeeker) error {
	for {
		name, size, err := readAtomHeader(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch name {
		case "meta":
			// next_item_id (int32)
			_, err := readBytes(r, 4)
			if err != nil {
				return err
			}
			fallthrough

		case "moov", "udta", "ilst":
			return m.readAtoms(r)
		}

		_, ok := atoms[name]
		var data []string
		if name == "----" {
			name, data, err = readCustomAtom(r, size)
			if err != nil {
				return err
			}

			if name != "----" {
				ok = true
				size = 0 // already read data
			}
		}

		if !ok {
			_, err := r.Seek(int64(size-8), io.SeekCurrent)
			if err != nil {
				return err
			}
			continue
		}

		err = m.readAtomData(r, name, size-8, data)
		if err != nil {
			return err
		}
	}
}

func (m metadataMP4) readAtomData(r io.ReadSeeker, name string, size uint32, processedData []string) error {
	var b []byte
	var err error
	var contentType string
	if len(processedData) > 0 {
		b = []byte(strings.Join(processedData, ";")) // add delimiter if multiple data fields
		contentType = "text"
	} else {
		// read the data
		b, err = readBytes(r, uint(size))
		if err != nil {
			return err
		}

		// covr may hold multiple "data" atoms, one per cover.
		if name == "covr" {
			m.data[name] = parseCoverArt(b)
			return nil
		}

		if len(b) < 8 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, got %d", 8, len(b))
		}

		// "data" + size (4 bytes each)
		b = b[8:]

		if len(b) < 4 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for class, got %d", 4, len(b))
		}
		class := getInt(b[1:4])
		var ok bool
		contentType, ok = atomTypes[class]
		if !ok {
			return fmt.Errorf("invalid content type: %v (%x) (%x)", class, b[1:4], b)
		}

		// 4: atom version (1 byte) + atom flags (3 bytes)
		// 4: NULL (usually locale indicator)
		if len(b) < 8 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for atom version and flags, got %d", 8, len(b))
		}
		b = b[8:]
	}

	if name == "trkn" || name == "disk" {
		if len(b) < 6 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for track and disk numbers, got %d", 6, len(b))
		}

		m.data[name] = int(b[3])
		m.data[name+"_count"] = int(b[5])
		return nil
	}

	var data interface{}
	switch contentType {
	case "implicit":
		if _, ok := atoms[name]; ok {
			return fmt.Errorf("unhandled implicit content type for required atom: %q", name)
		}
		return nil

	case "text":
		data = string(b)

	case "uint8":
		if len(b) < 1 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for integer tag data, got %d", 1, len(b))
		}
		data = getInt(b[:1])
	}
	m.data[name] = data

	return nil
}

// parseCoverArt splits a covr atom body into its "data" sub-atoms and returns a
// picture for each. The image format is sniffed from the bytes, falling back to
// the atom type code (13=jpeg, 14=png) when the bytes are not recognised.
func parseCoverArt(raw []byte) []*Picture {
	var pics []*Picture
	for len(raw) >= 16 {
		atomSize := int(binary.BigEndian.Uint32(raw[0:4]))
		if atomSize < 16 || atomSize > len(raw) {
			atomSize = len(raw)
		}

		if string(raw[4:8]) == "data" {
			var declaredMIME string
			switch atomTypes[getInt(raw[9:12])] {
			case "jpeg", "png":
				declaredMIME = "image/" + atomTypes[getInt(raw[9:12])]
			}
			data := raw[16:atomSize]
			mimeType, ext := resolveImageType(declaredMIME, data)
			pics = append(pics, &Picture{
				Ext:      ext,
				MIMEType: mimeType,
				Data:     data,
			})
		}

		raw = raw[atomSize:]
	}
	return pics
}

func readAtomHeader(r io.ReadSeeker) (name string, size uint32, err error) {
	err = binary.Read(r, binary.BigEndian, &size)
	if err != nil {
		return
	}
	name, err = readString(r, 4)
	return
}

// Generic atom.
// Should have 3 sub atoms : mean, name and data.
// We check that mean is "com.apple.iTunes" or others and we use the subname as
// the name, and move to the data atom.
// Data atom could have multiple data values, each with a header.
// If anything goes wrong, we jump at the end of the "----" atom.
func readCustomAtom(r io.ReadSeeker, size uint32) (_ string, data []string, _ error) {
	subNames := make(map[string]string)

	for size > 8 {
		subName, subSize, err := readAtomHeader(r)
		if err != nil {
			return "", nil, err
		}

		// Remove the size of the atom from the size counter
		if size >= subSize {
			size -= subSize
		} else {
			return "", nil, errors.New("--- invalid size")
		}

		b, err := readBytes(r, uint(subSize-8))
		if err != nil {
			return "", nil, err
		}

		if len(b) < 4 {
			return "", nil, fmt.Errorf("invalid encoding: expected at least %d bytes, got %d", 4, len(b))
		}
		switch subName {
		case "mean", "name":
			subNames[subName] = string(b[4:])
		case "data":
			data = append(data, string(b[4:]))
		}
	}

	// there should remain only the header size
	if size != 8 {
		err := errors.New("---- atom out of bounds")
		return "", nil, err
	}

	if !means[subNames["mean"]] || subNames["name"] == "" || len(data) == 0 {
		return "----", nil, nil
	}

	return subNames["name"], data, nil
}

func (metadataMP4) Format() Format       { return MP4 }
func (m metadataMP4) FileType() FileType { return m.fileType }

func (m metadataMP4) Raw() map[string]interface{} { return m.data }

func (m metadataMP4) getString(n []string) string {
	for _, k := range n {
		if x, ok := m.data[k]; ok {
			return x.(string)
		}
	}
	return ""
}

func (m metadataMP4) getInt(n []string) int {
	for _, k := range n {
		if x, ok := m.data[k]; ok {
			return x.(int)
		}
	}
	return 0
}

func (m metadataMP4) Title() string {
	return m.getString(atoms.Name("title"))
}

func (m metadataMP4) Artist() string {
	return m.getString(atoms.Name("artist"))
}

func (m metadataMP4) Album() string {
	return m.getString(atoms.Name("album"))
}

func (m metadataMP4) AlbumArtist() string {
	return m.getString(atoms.Name("album_artist"))
}

func (m metadataMP4) Composer() string {
	return m.getString(atoms.Name("composer"))
}

func (m metadataMP4) Genre() string {
	return m.getString(atoms.Name("genre"))
}

func (m metadataMP4) Year() int {
	date := m.getString(atoms.Name("year"))
	if len(date) >= 4 {
		year, _ := strconv.Atoi(date[:4])
		return year
	}
	return 0
}

func (m metadataMP4) Track() (int, int) {
	x := m.getInt([]string{"trkn"})
	if n, ok := m.data["trkn_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Disc() (int, int) {
	x := m.getInt([]string{"disk"})
	if n, ok := m.data["disk_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Lyrics() string {
	t, ok := m.data["\xa9lyr"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) Comment() string {
	t, ok := m.data["\xa9cmt"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) coverArt() []*Picture {
	v, ok := m.data["covr"]
	if !ok {
		return nil
	}
	pics, _ := v.([]*Picture)
	return pics
}

func (m metadataMP4) Picture() *Picture {
	pics := m.coverArt()
	if len(pics) == 0 {
		return nil
	}
	return pics[0]
}

// Pictures returns the attached cover art. The MP4 "covr" atom carries no
// picture type, so the returned pictures are untyped.
func (m metadataMP4) Pictures() []Picture {
	src := m.coverArt()
	if len(src) == 0 {
		return nil
	}
	pics := make([]Picture, len(src))
	for i, p := range src {
		pics[i] = *p
	}
	return pics
}
