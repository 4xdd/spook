package neural

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

const (
	weightMagic   = "MERT"
	weightVersion = 1
	dtypeF32      = 0
	dtypeI8       = 1
)

// Weights holds all tensors for a MERT checkpoint.
type Weights struct {
	Config  Config
	Tensors map[string]*Tensor
}

// Tensor is a dense row-major float32 array.
type Tensor struct {
	Dims   []int
	Data   []float32
	quant  bool
	scales []float32
}

func (t *Tensor) Size() int {
	n := 1
	for _, d := range t.Dims {
		n *= d
	}
	return n
}

func (t *Tensor) At(idx ...int) float32 {
	return t.Data[t.offset(idx)]
}

func (t *Tensor) offset(idx []int) int {
	off := 0
	stride := 1
	for i := len(t.Dims) - 1; i >= 0; i-- {
		off += idx[i] * stride
		stride *= t.Dims[i]
	}
	return off
}

// LoadWeights reads a .mert file produced by scripts/convert_mert.py.
func LoadWeights(path string) (*Weights, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readWeights(f)
}

func readWeights(r io.Reader) (*Weights, error) {
	var hdr [32]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if string(hdr[:4]) != weightMagic {
		return nil, fmt.Errorf("invalid magic %q", hdr[:4])
	}
	version := binary.LittleEndian.Uint32(hdr[4:8])
	if version != weightVersion {
		return nil, fmt.Errorf("unsupported version %d", version)
	}
	configLen := binary.LittleEndian.Uint32(hdr[8:12])
	numTensors := binary.LittleEndian.Uint32(hdr[12:16])

	cfgRaw := make([]byte, configLen)
	if _, err := io.ReadFull(r, cfgRaw); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg, err := parseConfig(cfgRaw)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	tensors := make(map[string]*Tensor, numTensors)
	for i := uint32(0); i < numTensors; i++ {
		var nameLen uint16
		if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
			return nil, err
		}
		nameBuf := make([]byte, nameLen)
		if _, err := io.ReadFull(r, nameBuf); err != nil {
			return nil, err
		}
		name := string(nameBuf)

		var ndim uint8
		if err := binary.Read(r, binary.LittleEndian, &ndim); err != nil {
			return nil, err
		}
		var dimsRaw [4]uint32
		if err := binary.Read(r, binary.LittleEndian, &dimsRaw); err != nil {
			return nil, err
		}
		dims := make([]int, ndim)
		for j := range dims {
			dims[j] = int(dimsRaw[j])
		}

		var dtype uint8
		if err := binary.Read(r, binary.LittleEndian, &dtype); err != nil {
			return nil, err
		}
		var scaleCount uint32
		if err := binary.Read(r, binary.LittleEndian, &scaleCount); err != nil {
			return nil, err
		}
		scales := make([]float32, scaleCount)
		if scaleCount > 0 {
			if err := binary.Read(r, binary.LittleEndian, &scales); err != nil {
				return nil, err
			}
		}

		var dataLen uint32
		if err := binary.Read(r, binary.LittleEndian, &dataLen); err != nil {
			return nil, err
		}

		t := &Tensor{Dims: dims}
		switch dtype {
		case dtypeF32:
			raw := make([]float32, dataLen)
			if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
				return nil, err
			}
			t.Data = raw
		case dtypeI8:
			raw := make([]int8, dataLen)
			if err := binary.Read(r, binary.LittleEndian, &raw); err != nil {
				return nil, err
			}
			t.quant = true
			t.scales = scales
			t.Data = dequantInt8(raw, scales, dims)
		default:
			return nil, fmt.Errorf("tensor %s: unknown dtype %d", name, dtype)
		}
		tensors[name] = t
	}

	return &Weights{Config: cfg, Tensors: tensors}, nil
}

func dequantInt8(raw []int8, scales []float32, dims []int) []float32 {
	if len(dims) == 0 {
		return nil
	}
	outDim := dims[0]
	inner := 1
	for _, d := range dims[1:] {
		inner *= d
	}
	out := make([]float32, len(raw))
	for o := 0; o < outDim; o++ {
		scale := scales[o] / 127.0
		base := o * inner
		for i := 0; i < inner; i++ {
			out[base+i] = float32(raw[base+i]) * scale
		}
	}
	return out
}

// ConfigJSON returns the embedded config for debugging.
func (w *Weights) ConfigJSON() string {
	b, _ := json.Marshal(w.Config)
	return string(b)
}

func requireTensor(w *Weights, name string) (*Tensor, error) {
	t, ok := w.Tensors[name]
	if !ok {
		return nil, fmt.Errorf("missing tensor %q", name)
	}
	return t, nil
}

func nearlyEqual(a, b []float32, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(float64(a[i]-b[i])) > tol {
			return false
		}
	}
	return true
}
