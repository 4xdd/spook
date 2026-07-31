package audio

import (
	"io"
)

// probeMP4 walks the ISO base media box tree for mvhd (movie duration) and the
// audio track's mdhd (whose timescale is the sample rate).
func probeMP4(r io.ReadSeeker) (Info, error) {
	size := fileSize(r)
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}

	var info Info
	found := false

	var walk func(start, end int64, depth int) error
	walk = func(start, end int64, depth int) error {
		if depth > 6 {
			return nil
		}
		offset := start
		header := make([]byte, 8)
		for offset+8 <= end {
			if _, err := r.Seek(offset, io.SeekStart); err != nil {
				return err
			}
			if _, err := io.ReadFull(r, header); err != nil {
				return nil
			}
			boxSize := int64(be32(header[0:4]))
			boxType := string(header[4:8])
			headerLen := int64(8)

			if boxSize == 1 {
				ext := make([]byte, 8)
				if _, err := io.ReadFull(r, ext); err != nil {
					return nil
				}
				boxSize = int64(be64(ext))
				headerLen = 16
			} else if boxSize == 0 {
				boxSize = end - offset
			}
			if boxSize < headerLen || offset+boxSize > end {
				return nil
			}

			switch boxType {
			case "moov", "trak", "mdia":
				if err := walk(offset+headerLen, offset+boxSize, depth+1); err != nil {
					return err
				}
			case "mvhd", "mdhd":
				body := make([]byte, min(int(boxSize-headerLen), 32))
				if _, err := io.ReadFull(r, body); err == nil {
					if timescale, duration, ok := parseTimeBox(body); ok {
						ms := duration * 1000 / timescale
						if boxType == "mdhd" {
							// The media header belongs to the audio track, so its
							// timescale is the sample rate.
							info.SampleRateHz = int(timescale)
							info.DurationMS = ms
							found = true
						} else if !found {
							info.DurationMS = ms
							found = true
						}
					}
				}
			}

			offset += boxSize
		}
		return nil
	}

	if err := walk(0, size, 0); err != nil {
		return info, err
	}
	if info.DurationMS <= 0 {
		return info, errUnsupported
	}
	return info, nil
}

func parseTimeBox(b []byte) (timescale, duration int64, ok bool) {
	if len(b) < 4 {
		return 0, 0, false
	}
	version := b[0]
	switch version {
	case 0:
		if len(b) < 20 {
			return 0, 0, false
		}
		timescale = int64(be32(b[12:16]))
		duration = int64(be32(b[16:20]))
	case 1:
		if len(b) < 32 {
			return 0, 0, false
		}
		timescale = int64(be32(b[20:24]))
		duration = int64(be64(b[24:32]))
	default:
		return 0, 0, false
	}
	if timescale == 0 || duration == 0 {
		return 0, 0, false
	}
	return timescale, duration, true
}
