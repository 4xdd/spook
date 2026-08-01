package neural

// modelBuffers holds reusable forward-pass scratch (per Model, not shared).
type modelBuffers struct {
	q, k, v, merged []float32
	scores            []float32
	ffn               []float32
}

func (m *Model) ensureBuf(seqLen, dim int) {
	need := seqLen * dim
	if cap(m.buf.q) < need {
		m.buf.q = make([]float32, need)
		m.buf.k = make([]float32, need)
		m.buf.v = make([]float32, need)
		m.buf.merged = make([]float32, need)
	}
	scoreNeed := seqLen * seqLen
	if cap(m.buf.scores) < scoreNeed {
		m.buf.scores = make([]float32, scoreNeed)
	}
	ffnNeed := seqLen * m.cfg.IntermediateSize
	if cap(m.buf.ffn) < ffnNeed {
		m.buf.ffn = make([]float32, ffnNeed)
	}
}

func (m *Model) qBuf(seqLen, dim int) []float32 { return m.buf.q[:seqLen*dim] }
func (m *Model) kBuf(seqLen, dim int) []float32 { return m.buf.k[:seqLen*dim] }
func (m *Model) vBuf(seqLen, dim int) []float32 { return m.buf.v[:seqLen*dim] }
func (m *Model) mergedBuf(seqLen, dim int) []float32 {
	return m.buf.merged[:seqLen*dim]
}
func (m *Model) scoresBuf(seqLen int) []float32 { return m.buf.scores[:seqLen*seqLen] }
func (m *Model) ffnBuf(seqLen int) []float32 {
	return m.buf.ffn[:seqLen*m.cfg.IntermediateSize]
}
