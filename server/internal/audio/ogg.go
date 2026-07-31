package audio

import (
	"bytes"
	"io"
)

// probeOgg reads the codec identification header from the first page and the
// granule position from the last, which together give an exact duration.
func probeOgg(r io.ReadSeeker) (Info, error) {
	size := fileSize(r)
	if size < 64 {
		return Info{}, errUnsupported
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}
	head := make([]byte, min(int(size), 8192))
	if _, err := io.ReadFull(r, head); err != nil {
		return Info{}, err
	}
	if !bytes.HasPrefix(head, []byte("OggS")) {
		return Info{}, errUnsupported
	}

	info, granuleRate, ok := parseOggIdentification(head)
	if !ok {
		return Info{}, errUnsupported
	}

	tailLen := int64(min(int(size), 65536))
	if _, err := r.Seek(size-tailLen, io.SeekStart); err != nil {
		return Info{}, err
	}
	tail := make([]byte, tailLen)
	if _, err := io.ReadFull(r, tail); err != nil {
		return Info{}, err
	}

	index := bytes.LastIndex(tail, []byte("OggS"))
	if index < 0 || index+14 > len(tail) {
		return info, errUnsupported
	}
	granule := int64(le64(tail[index+6 : index+14]))
	if granule <= 0 || granuleRate <= 0 {
		return info, errUnsupported
	}

	info.DurationMS = granule * 1000 / int64(granuleRate)
	return info, nil
}

// parseOggIdentification returns stream info plus the rate that granule
// positions are counted in (always 48kHz for Opus).
func parseOggIdentification(head []byte) (Info, int, bool) {
	if index := bytes.Index(head, []byte("OpusHead")); index >= 0 && index+19 <= len(head) {
		body := head[index:]
		// The header records the pre-encode input rate; Opus itself always
		// decodes at 48kHz, which is the rate granule positions count in.
		return Info{Channels: int(body[9]), SampleRateHz: 48000}, 48000, true
	}

	if index := bytes.Index(head, []byte("\x01vorbis")); index >= 0 && index+30 <= len(head) {
		body := head[index:]
		info := Info{
			Channels:     int(body[11]),
			SampleRateHz: int(le32(body[12:16])),
		}
		if nominal := int(le32(body[20:24])); nominal > 0 {
			info.BitrateKbps = nominal / 1000
		}
		if info.SampleRateHz == 0 {
			return info, 0, false
		}
		return info, info.SampleRateHz, true
	}

	if index := bytes.Index(head, []byte("\x7fFLAC")); index >= 0 && index+13+18 <= len(head) {
		// The Ogg mapping embeds a native STREAMINFO block after a 13 byte header.
		if info, err := parseStreamInfo(head[index+13+4:]); err == nil && info.SampleRateHz > 0 {
			return Info{SampleRateHz: info.SampleRateHz, Channels: info.Channels}, info.SampleRateHz, true
		}
	}

	return Info{}, 0, false
}
