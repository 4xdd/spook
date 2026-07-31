package audio

import (
	"bytes"
	"io"
)

var mp3Bitrates = map[int][]int{
	// MPEG 1
	31: {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0},
	32: {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0},
	33: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
	// MPEG 2 / 2.5
	21: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0},
	22: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
	23: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
}

var mp3SampleRates = map[int][]int{
	3: {44100, 48000, 32000}, // MPEG 1
	2: {22050, 24000, 16000}, // MPEG 2
	0: {11025, 12000, 8000},  // MPEG 2.5
}

type mp3Frame struct {
	mpegVersion int // 3 = MPEG1, 2 = MPEG2, 0 = MPEG2.5
	layer       int // 1..3
	bitrate     int // kbps
	sampleRate  int
	channels    int
	samples     int
	frameLen    int
	mono        bool
}

// probeMP3 prefers the VBR header's frame count and falls back to a
// constant-bitrate estimate over the remaining file length.
func probeMP3(r io.ReadSeeker) (Info, error) {
	size := fileSize(r)
	start := skipID3(r)
	if _, err := r.Seek(start, io.SeekStart); err != nil {
		return Info{}, err
	}

	buf := make([]byte, 16*1024)
	n, err := io.ReadFull(r, buf)
	if n == 0 {
		return Info{}, err
	}
	buf = buf[:n]

	offset, frame, ok := findFrame(buf)
	if !ok {
		return Info{}, errUnsupported
	}

	info := Info{
		SampleRateHz: frame.sampleRate,
		Channels:     frame.channels,
		BitrateKbps:  frame.bitrate,
	}

	if frames, bytesLen, ok := vbrFrameCount(buf[offset:], frame); ok && frames > 0 {
		info.DurationMS = int64(frames) * int64(frame.samples) * 1000 / int64(frame.sampleRate)
		if bytesLen > 0 && info.DurationMS > 0 {
			info.BitrateKbps = int(int64(bytesLen) * 8 / info.DurationMS)
		}
		return info, nil
	}

	audioBytes := size - start - int64(offset)
	if frame.bitrate > 0 && audioBytes > 0 {
		info.DurationMS = audioBytes * 8 / int64(frame.bitrate)
	}
	if info.DurationMS == 0 {
		return info, errUnsupported
	}
	return info, nil
}

func findFrame(buf []byte) (int, mp3Frame, bool) {
	for i := 0; i+4 <= len(buf); i++ {
		if buf[i] != 0xff || buf[i+1]&0xe0 != 0xe0 {
			continue
		}
		frame, ok := parseFrameHeader(buf[i : i+4])
		if !ok {
			continue
		}
		// Confirm with the next frame header where the file is long enough,
		// which rejects sync patterns inside stray binary data.
		next := i + frame.frameLen
		if next+4 <= len(buf) {
			if _, ok := parseFrameHeader(buf[next : next+4]); !ok {
				continue
			}
		}
		return i, frame, true
	}
	return 0, mp3Frame{}, false
}

func parseFrameHeader(b []byte) (mp3Frame, bool) {
	versionBits := int(b[1]>>3) & 0x03
	layerBits := int(b[1]>>1) & 0x03
	bitrateIndex := int(b[2]>>4) & 0x0f
	sampleRateIndex := int(b[2]>>2) & 0x03
	padding := int(b[2]>>1) & 0x01
	channelMode := int(b[3]>>6) & 0x03

	if versionBits == 1 || layerBits == 0 || bitrateIndex == 0 || bitrateIndex == 15 || sampleRateIndex == 3 {
		return mp3Frame{}, false
	}

	layer := 4 - layerBits
	version := versionBits
	rateKey := version

	tableKey := 3*10 + layer
	if version != 3 {
		tableKey = 2*10 + layer
	}
	bitrates, ok := mp3Bitrates[tableKey]
	if !ok {
		return mp3Frame{}, false
	}
	rates, ok := mp3SampleRates[rateKey]
	if !ok {
		return mp3Frame{}, false
	}

	frame := mp3Frame{
		mpegVersion: version,
		layer:       layer,
		bitrate:     bitrates[bitrateIndex],
		sampleRate:  rates[sampleRateIndex],
		mono:        channelMode == 3,
	}
	if frame.bitrate == 0 || frame.sampleRate == 0 {
		return mp3Frame{}, false
	}
	frame.channels = 2
	if frame.mono {
		frame.channels = 1
	}

	switch {
	case frame.layer == 1:
		frame.samples = 384
	case frame.layer == 2:
		frame.samples = 1152
	case frame.mpegVersion == 3:
		frame.samples = 1152
	default:
		frame.samples = 576
	}

	if frame.layer == 1 {
		frame.frameLen = (12*frame.bitrate*1000/frame.sampleRate + padding) * 4
	} else {
		frame.frameLen = frame.samples/8*frame.bitrate*1000/frame.sampleRate + padding
	}
	if frame.frameLen <= 4 {
		return mp3Frame{}, false
	}

	return frame, true
}

// vbrFrameCount reads a Xing/Info or VBRI header from inside the first frame.
func vbrFrameCount(frameData []byte, frame mp3Frame) (int, int, bool) {
	offset := 36
	switch {
	case frame.mpegVersion == 3 && frame.mono:
		offset = 21
	case frame.mpegVersion != 3 && frame.mono:
		offset = 13
	case frame.mpegVersion != 3:
		offset = 21
	}

	if offset+12 <= len(frameData) {
		tag := frameData[offset : offset+4]
		if bytes.Equal(tag, []byte("Xing")) || bytes.Equal(tag, []byte("Info")) {
			flags := be32(frameData[offset+4 : offset+8])
			cursor := offset + 8
			frames, byteCount := 0, 0
			if flags&0x01 != 0 && cursor+4 <= len(frameData) {
				frames = int(be32(frameData[cursor : cursor+4]))
				cursor += 4
			}
			if flags&0x02 != 0 && cursor+4 <= len(frameData) {
				byteCount = int(be32(frameData[cursor : cursor+4]))
			}
			return frames, byteCount, frames > 0
		}
	}

	const vbriOffset = 36
	if vbriOffset+26 <= len(frameData) && bytes.Equal(frameData[vbriOffset:vbriOffset+4], []byte("VBRI")) {
		byteCount := int(be32(frameData[vbriOffset+10 : vbriOffset+14]))
		frames := int(be32(frameData[vbriOffset+14 : vbriOffset+18]))
		return frames, byteCount, frames > 0
	}

	return 0, 0, false
}
