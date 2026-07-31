package audio

import (
	"io"
)

// probeWAV walks the RIFF chunk list for fmt and data.
func probeWAV(r io.ReadSeeker) (Info, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return Info{}, err
	}

	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return Info{}, err
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return Info{}, errUnsupported
	}

	var info Info
	var byteRate uint32
	var dataSize uint32

	chunk := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, chunk); err != nil {
			break
		}
		id := string(chunk[0:4])
		size := le32(chunk[4:8])

		switch id {
		case "fmt ":
			body := make([]byte, min(int(size), 16))
			if _, err := io.ReadFull(r, body); err != nil {
				return info, err
			}
			if len(body) >= 16 {
				info.Channels = int(le16(body[2:4]))
				info.SampleRateHz = int(le32(body[4:8]))
				byteRate = le32(body[8:12])
			}
			if remaining := int64(size) - int64(len(body)); remaining > 0 {
				if _, err := r.Seek(remaining, io.SeekCurrent); err != nil {
					return info, err
				}
			}
		case "data":
			dataSize = size
			if _, err := r.Seek(int64(size), io.SeekCurrent); err != nil {
				// A truncated data chunk still tells us the duration.
				break
			}
		default:
			if _, err := r.Seek(int64(size), io.SeekCurrent); err != nil {
				break
			}
		}

		// RIFF chunks are word aligned.
		if size%2 == 1 {
			if _, err := r.Seek(1, io.SeekCurrent); err != nil {
				break
			}
		}
		if dataSize > 0 && byteRate > 0 {
			break
		}
	}

	if byteRate > 0 && dataSize > 0 {
		info.DurationMS = int64(dataSize) * 1000 / int64(byteRate)
		info.BitrateKbps = int(byteRate * 8 / 1000)
	}
	if info.DurationMS == 0 {
		return info, errUnsupported
	}
	return info, nil
}
