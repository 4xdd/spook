package audio

import (
	"io"
)

// probeFLAC reads STREAMINFO, which carries an exact sample count.
func probeFLAC(r io.ReadSeeker) (Info, error) {
	offset := skipID3(r)
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return Info{}, err
	}

	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return Info{}, err
	}
	if string(magic) != "fLaC" {
		return Info{}, errUnsupported
	}

	header := make([]byte, 4)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			return Info{}, err
		}
		last := header[0]&0x80 != 0
		blockType := header[0] & 0x7f
		length := int(header[1])<<16 | int(header[2])<<8 | int(header[3])

		if blockType == 0 {
			block := make([]byte, length)
			if _, err := io.ReadFull(r, block); err != nil {
				return Info{}, err
			}
			return parseStreamInfo(block)
		}
		if last {
			return Info{}, errUnsupported
		}
		if _, err := r.Seek(int64(length), io.SeekCurrent); err != nil {
			return Info{}, err
		}
	}
}

func parseStreamInfo(b []byte) (Info, error) {
	if len(b) < 18 {
		return Info{}, errUnsupported
	}

	sampleRate := int(b[10])<<12 | int(b[11])<<4 | int(b[12])>>4
	channels := int((b[12]>>1)&0x07) + 1
	totalSamples := int64(b[13]&0x0f)<<32 | int64(b[14])<<24 |
		int64(b[15])<<16 | int64(b[16])<<8 | int64(b[17])

	info := Info{SampleRateHz: sampleRate, Channels: channels}
	if sampleRate > 0 && totalSamples > 0 {
		info.DurationMS = totalSamples * 1000 / int64(sampleRate)
	}
	return info, nil
}
