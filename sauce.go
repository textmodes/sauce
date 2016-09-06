// Package sauce contains a SAUCE (Standard Architecture for Universal Comment Extensions) parser.
package sauce

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

const (
	// ASCIISub is the SUB ASCII character (or EOF)
	ASCIISub = '\x1a'

	sauceDate = "19700101"
)

const (
	// LetterSpacingLegacy enables legacy letter spacing
	LetterSpacingLegacy = iota
	// LetterSpacing8Pixel enables 8 pixel letter spacing
	LetterSpacing8Pixel
	// LetterSpacing9Pixel enables 9 pixel letter spacing
	LetterSpacing9Pixel
	// LetterSpacingInvalid is unspecified
	LetterSpacingInvalid
)

const (
	// AspectRatioLegacy enables legacy aspect ratio
	AspectRatioLegacy = iota
	// AspectRatioStretch enables stretching on displays with square pixels
	AspectRatioStretch
	// AspectRatioSquare enables optimization for non-square displays
	AspectRatioSquare
	// AspectRatioInvalid is unspecified
	AspectRatioInvalid
)

var (
	// ID is the SAUCE header identifier
	ID = [5]byte{'S', 'A', 'U', 'C', 'E'}

	// Version is the SAUCE version
	Version = [2]byte{0, 0}

	errShortRead = errors.New("Short read")
)

// SAUCE (Standard Architecture for Universal Comment Extensions) record.
type SAUCE struct {
	ID       [5]byte
	Version  [2]byte
	Title    string
	Author   string
	Group    string
	Date     time.Time
	FileSize uint32
	DataType uint8
	FileType uint8
	TInfo    [4]uint16
	Comments uint8
	TFlags   TFlags
	TInfos   []byte
}

// TFlags contains a parsed TFlags structure
type TFlags struct {
	NonBlink      bool
	LetterSpacing uint8
	AspectRatio   uint8
}

// New creates an empty SAUCE record.
func New() *SAUCE {
	return &SAUCE{
		ID:      ID,
		Version: Version,
		TInfo:   [4]uint16{},
		TInfos:  []byte{},
		TFlags:  TFlags{},
	}
}

// Parse SAUCE record
func Parse(s *io.SectionReader) (r *SAUCE, err error) {
	var n int64
	var i int

	n, err = s.Seek(-128, 2)
	if err != nil {
		return
	}
	if n < 128 {
		return nil, errShortRead
	}

	b := make([]byte, 128)
	i, err = s.Read(b)
	if err != nil {
		return
	}
	if i != 128 {
		return nil, errShortRead
	}
	return ParseBytes(b)
}

// ParseReader reads the SAUCE header from a stream
func ParseReader(i io.Reader) (r *SAUCE, err error) {
	var b []byte
	if b, err = ioutil.ReadAll(i); err != nil {
		return nil, err
	}
	if len(b) < 128 {
		return nil, errShortRead
	}
	return ParseBytes(b)
}

// ParseBytes reads the SAUCE header from a slice of bytes
func ParseBytes(b []byte) (r *SAUCE, err error) {
	if len(b) < 128 {
		return nil, errors.New("Short read")
	}
	o := len(b) - 128
	if !bytes.Equal(b[o+0:o+5], ID[:]) {
		return nil, errors.New("No SAUCE record")
	}

	r = New()
	r.Title = strings.TrimSpace(string(b[o+7 : o+41]))
	r.Author = strings.TrimSpace(string(b[o+41 : o+61]))
	r.Group = strings.TrimSpace(string(b[o+61 : o+81]))
	r.Date = r.parseDate(string(b[o+82 : o+90]))
	r.FileSize = binary.LittleEndian.Uint32(b[o+91 : o+95])
	r.DataType = uint8(b[o+94])
	r.FileType = uint8(b[o+95])
	r.TInfo[0] = binary.LittleEndian.Uint16(b[o+96 : o+98])
	r.TInfo[1] = binary.LittleEndian.Uint16(b[o+98 : o+100])
	r.TInfo[2] = binary.LittleEndian.Uint16(b[o+100 : o+102])
	r.TInfo[3] = binary.LittleEndian.Uint16(b[o+102 : o+104])
	r.Comments = uint8(b[o+104])
	r.TInfos = b[106+o:]
	tflags := uint8(b[o+105])
	r.TFlags.NonBlink = (tflags & 1) == 1
	r.TFlags.LetterSpacing = (tflags >> 1) & 3
	r.TFlags.AspectRatio = (tflags >> 3) & 3
	return r, nil
}

func (r *SAUCE) parseDate(s string) time.Time {
	y, _ := strconv.Atoi(s[:4])
	m, _ := strconv.Atoi(s[4:6])
	d, _ := strconv.Atoi(s[6:8])
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

// Dump the contents of the SAUCE record to stdout.
func (r *SAUCE) Dump() {
	fmt.Printf("id......: %s\n", string(r.ID[:]))
	fmt.Printf("version.: %d%d\n", r.Version[0], r.Version[1])
	fmt.Printf("title...: %s\n", r.Title)
	fmt.Printf("author..: %s\n", r.Author)
	fmt.Printf("group...: %s\n", r.Group)
	fmt.Printf("date....: %s\n", r.Date)
	fmt.Printf("filesize: %d\n", r.FileSize)
	fmt.Printf("datatype: %d (%s)\n", r.DataType, r.DataTypeString())
	if FileType[r.DataType] != nil {
		fmt.Printf("filetype: %d (%s)\n", r.FileType, r.FileTypeString())
	} else {
		fmt.Printf("filetype: %d\n", r.FileType)
	}
	fmt.Printf("tinfo...: %d, %d, %d, %d\n", r.TInfo[0], r.TInfo[1], r.TInfo[2], r.TInfo[3])
	switch r.DataType {
	case 1:
		switch r.FileType {
		case 0, 1, 2, 4, 5, 8:
			w := r.TInfo[0]
			h := r.TInfo[1]
			if w == 0 {
				w = 80
			}
			fmt.Printf("size....: %d x %d characters\n", w, h)
		case 3:
			fmt.Printf("size....: %d x %d pixels\n", r.TInfo[0], r.TInfo[1])
		}
	case 2:
		fmt.Printf("size....: %d x %d pixels\n", r.TInfo[0], r.TInfo[1])
	}
}

// DataTypeString returns the DataType as string.
func (r *SAUCE) DataTypeString() string {
	return DataType[r.DataType]
}

// FileTypeString returns the FileType as string.
func (r *SAUCE) FileTypeString() string {
	switch FileType[r.DataType] {
	case nil:
		switch r.DataType {
		case DataTypeBinaryText:
			return "BinaryText"
		case DataTypeXBIN:
			return "XBin"
		case DataTypeExecutable:
			return "Executable"
		}
	default:
		return FileType[r.DataType][r.FileType]
	}

	return ""
}

// MimeType returns the mime type as string.
func (r *SAUCE) MimeType() (t string) {
	switch MimeType[r.DataType] {
	case nil:
		switch r.DataType {
		case DataTypeBinaryText:
			t = "text/x-binary"
		case DataTypeXBIN:
			t = "text/x-xbin"
		}
	default:
		t = MimeType[r.DataType][r.FileType]
	}
	if t == "" {
		t = "application/octet-stream"
	}
	return
}

// Font returns the font name
func (r *SAUCE) Font() string {
	return strings.Trim(string(r.TInfos[:]), "\x00 ")
}
