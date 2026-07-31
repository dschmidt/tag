// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	vorbisCommentPrefix = []byte("\x03vorbis")
	opusTagsPrefix      = []byte("OpusTags")
	speexPrefix         = []byte("Speex   ")
	theoraCommentPrefix = []byte("\x81theora")
	flacInOggPrefix     = []byte("\x7fFLAC")
)

var oggCRC32Poly04c11db7 = oggCRCTable(0x04c11db7)

type crc32Table [256]uint32

func oggCRCTable(poly uint32) *crc32Table {
	var t crc32Table

	for i := 0; i < 256; i++ {
		crc := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		t[i] = crc
	}

	return &t
}

func oggCRCUpdate(crc uint32, tab *crc32Table, p []byte) uint32 {
	for _, v := range p {
		crc = (crc << 8) ^ tab[byte(crc>>24)^v]
	}
	return crc
}

type oggPageHeader struct {
	Magic           [4]byte // "OggS"
	Version         uint8
	Flags           uint8
	GranulePosition uint64
	SerialNumber    uint32
	SequenceNumber  uint32
	CRC             uint32
	Segments        uint8
}

type oggDemuxer struct {
	packetBufs map[uint32]*bytes.Buffer
}

// Read ogg packets, can return empty slice of packets and nil err
// if more data is needed
func (o *oggDemuxer) Read(r io.Reader) ([][]byte, error) {
	headerBuf := &bytes.Buffer{}
	var oh oggPageHeader
	if err := binary.Read(io.TeeReader(r, headerBuf), binary.LittleEndian, &oh); err != nil {
		return nil, err
	}

	if bytes.Compare(oh.Magic[:], []byte("OggS")) != 0 {
		// TODO: seek for syncword?
		return nil, errors.New("expected 'OggS'")
	}

	segmentTable := make([]byte, oh.Segments)
	if _, err := io.ReadFull(r, segmentTable); err != nil {
		return nil, err
	}
	var segmentsSize int64
	for _, s := range segmentTable {
		segmentsSize += int64(s)
	}
	segmentsData := make([]byte, segmentsSize)
	if _, err := io.ReadFull(r, segmentsData); err != nil {
		return nil, err
	}

	headerBytes := headerBuf.Bytes()
	// reset CRC to zero in header before checksum
	headerBytes[22] = 0
	headerBytes[23] = 0
	headerBytes[24] = 0
	headerBytes[25] = 0
	crc := oggCRCUpdate(0, oggCRC32Poly04c11db7, headerBytes)
	crc = oggCRCUpdate(crc, oggCRC32Poly04c11db7, segmentTable)
	crc = oggCRCUpdate(crc, oggCRC32Poly04c11db7, segmentsData)
	if crc != oh.CRC {
		return nil, fmt.Errorf("expected crc %x != %x", oh.CRC, crc)
	}

	if o.packetBufs == nil {
		o.packetBufs = map[uint32]*bytes.Buffer{}
	}

	var packetBuf *bytes.Buffer
	continued := oh.Flags&0x1 != 0
	if continued {
		if b, ok := o.packetBufs[oh.SerialNumber]; ok {
			packetBuf = b
		} else {
			return nil, fmt.Errorf("could not find continued packet %d", oh.SerialNumber)
		}
	} else {
		packetBuf = &bytes.Buffer{}
	}

	var packets [][]byte
	var p int
	for _, s := range segmentTable {
		packetBuf.Write(segmentsData[p : p+int(s)])
		if s < 255 {
			packets = append(packets, packetBuf.Bytes())
			packetBuf = &bytes.Buffer{}
		}
		p += int(s)
	}

	o.packetBufs[oh.SerialNumber] = packetBuf

	return packets, nil
}

// ReadOGGTags reads OGG metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
// See http://www.xiph.org/vorbis/doc/Vorbis_I_spec.html
// and http://www.xiph.org/ogg/doc/framing.html for details.
// For Opus see https://tools.ietf.org/html/rfc7845
func ReadOGGTags(r io.Reader) (Metadata, error) {
	od := &oggDemuxer{}
	sawSpeex := false
	for {
		bs, err := od.Read(r)
		if err != nil {
			return nil, err
		}

		for _, b := range bs {
			var comment []byte
			switch {
			case bytes.HasPrefix(b, vorbisCommentPrefix):
				comment = b[len(vorbisCommentPrefix):]
			case bytes.HasPrefix(b, opusTagsPrefix):
				comment = b[len(opusTagsPrefix):]
			case bytes.HasPrefix(b, theoraCommentPrefix):
				comment = b[len(theoraCommentPrefix):]
			case bytes.HasPrefix(b, flacInOggPrefix):
				m := &metadataOGG{newMetadataVorbis()}
				err = m.readFLACInOgg(b, od, r)
				return m, err
			case bytes.HasPrefix(b, speexPrefix):
				// The Speex comment packet has no prefix; it follows the header.
				sawSpeex = true
				continue
			case sawSpeex:
				comment = b
			default:
				continue
			}

			m := &metadataOGG{newMetadataVorbis()}
			err = m.readVorbisComment(bytes.NewReader(comment))
			return m, err
		}
	}
}

// readFLACInOgg reads the FLAC metadata blocks carried by a FLAC-in-Ogg stream.
// The first packet holds the "\x7fFLAC" header plus STREAMINFO; each subsequent
// Ogg packet carries one further metadata block.
func (m *metadataOGG) readFLACInOgg(first []byte, od *oggDemuxer, r io.Reader) error {
	if len(first) < 13 || string(first[9:13]) != "fLaC" {
		return errors.New("invalid FLAC-in-Ogg header")
	}

	packets := [][]byte{first[13:]}
	for {
		for len(packets) == 0 {
			bs, err := od.Read(r)
			if err != nil {
				return err
			}
			packets = append(packets, bs...)
		}

		br := bytes.NewReader(packets[0])
		packets = packets[1:]
		for br.Len() > 0 {
			last, err := m.readFLACMetadataBlock(br)
			if err != nil {
				return err
			}
			if last {
				return nil
			}
		}
	}
}

type metadataOGG struct {
	*metadataVorbis
}

func (m *metadataOGG) FileType() FileType {
	return OGG
}
