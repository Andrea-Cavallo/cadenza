package session

import (
	"encoding/binary"
	"fmt"
)

const headerSize = 32

var headerMagic = [3]byte{'C', 'Z', 'S'}

const headerFileVersion = uint8(1)

// ExportedHeader è il header di una sessione .czs (esportato per i test).
type ExportedHeader struct {
	SaveReason   uint8
	MessageCount uint32
	PatternCount uint32
	CreatedAt    int64
	UpdatedAt    int64
}

// EncodeHeader serializza un ExportedHeader in 32 byte (little-endian).
func EncodeHeader(h ExportedHeader) [headerSize]byte {
	var buf [headerSize]byte
	buf[0] = headerMagic[0]
	buf[1] = headerMagic[1]
	buf[2] = headerMagic[2]
	buf[3] = headerFileVersion
	buf[4] = h.SaveReason
	// buf[5:8] riservati, zero
	binary.LittleEndian.PutUint32(buf[8:12], h.MessageCount)
	binary.LittleEndian.PutUint32(buf[12:16], h.PatternCount)
	binary.LittleEndian.PutUint64(buf[16:24], uint64(h.CreatedAt))
	binary.LittleEndian.PutUint64(buf[24:32], uint64(h.UpdatedAt))
	return buf
}

// DecodeHeader deserializza 32 byte in un ExportedHeader.
func DecodeHeader(buf [headerSize]byte) (ExportedHeader, error) {
	if buf[0] != headerMagic[0] || buf[1] != headerMagic[1] || buf[2] != headerMagic[2] {
		return ExportedHeader{}, fmt.Errorf("magic non valido: %q", buf[0:3])
	}
	if buf[3] != headerFileVersion {
		return ExportedHeader{}, fmt.Errorf("versione non supportata: %d", buf[3])
	}
	return ExportedHeader{
		SaveReason:   buf[4],
		MessageCount: binary.LittleEndian.Uint32(buf[8:12]),
		PatternCount: binary.LittleEndian.Uint32(buf[12:16]),
		CreatedAt:    int64(binary.LittleEndian.Uint64(buf[16:24])),
		UpdatedAt:    int64(binary.LittleEndian.Uint64(buf[24:32])),
	}, nil
}
