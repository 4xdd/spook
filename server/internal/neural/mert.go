package neural

import (
	"fmt"
	"math"
)

// Model runs MERT-v1-95M inference (HuBERT backbone without CQT).
type Model struct {
	cfg Config
	w   *Weights
}

func NewModel(w *Weights) (*Model, error) {
	if w == nil {
		return nil, fmt.Errorf("weights required")
	}
	if w.Config.FeatureExtractorCQT {
		return nil, fmt.Errorf("CQT feature extractor is not supported in native Go yet")
	}
	return &Model{cfg: w.Config, w: w}, nil
}

// Forward runs the encoder on mono float32 waveform at cfg.SampleRate.
// Returns frame embeddings [seqLen, hiddenSize].
func (m *Model) Forward(waveform []float32) ([]float32, int, error) {
	if len(waveform) == 0 {
		return nil, 0, fmt.Errorf("empty waveform")
	}

	feOut, feLen, err := m.featureExtractor(waveform)
	if err != nil {
		return nil, 0, err
	}

	proj, err := m.featureProjection(feOut, feLen)
	if err != nil {
		return nil, 0, err
	}

	hidden, seqLen, err := m.encoder(proj, feLen)
	if err != nil {
		return nil, 0, err
	}
	return hidden, seqLen, nil
}

// Embed mean-pools frame embeddings into a single track vector.
func (m *Model) Embed(waveform []float32) ([]float32, error) {
	hidden, seqLen, err := m.Forward(waveform)
	if err != nil {
		return nil, err
	}
	out := make([]float32, m.cfg.HiddenSize)
	meanPoolTime(out, hidden, seqLen, m.cfg.HiddenSize)
	return out, nil
}

func (m *Model) Config() Config { return m.cfg }

func (m *Model) featureExtractor(waveform []float32) ([]float32, int, error) {
	inLen := len(waveform)
	// Shape (1, T) -> (1, 1, T)
	x := make([]float32, inLen)
	copy(x, waveform)

	channels := 1
	length := inLen
	tmp := make([]float32, 0, channels*length*2)

	for layer := 0; layer < len(m.cfg.ConvDim); layer++ {
		outCh := m.cfg.ConvDim[layer]
		kernel := m.cfg.ConvKernel[layer]
		stride := m.cfg.ConvStride[layer]
		outLen := (length-kernel)/stride + 1
		if outLen < 1 {
			return nil, 0, fmt.Errorf("conv layer %d: invalid output length", layer)
		}

		w, err := requireTensor(m.w, fmt.Sprintf("feature_extractor.conv_layers.%d.conv.weight", layer))
		if err != nil {
			return nil, 0, err
		}

		out := make([]float32, outCh*outLen)
		conv1DOut(out, x, channels, outCh, length, kernel, stride, w)

		if layer == 0 && m.cfg.FeatExtractNorm == "group" {
			gamma, err := requireTensor(m.w, fmt.Sprintf("feature_extractor.conv_layers.%d.layer_norm.weight", layer))
			if err != nil {
				return nil, 0, err
			}
			beta, err := requireTensor(m.w, fmt.Sprintf("feature_extractor.conv_layers.%d.layer_norm.bias", layer))
			if err != nil {
				return nil, 0, err
			}
			groupNormChannels(out, out, outCh, outLen, gamma.Data, beta.Data, m.cfg.LayerNormEps)
		}

		geluSlice(out, out)

		tmp, x = x, tmp
		x = out
		channels = outCh
		length = outLen
	}

	// Return (channels, length) as flat [channels*length]
	return x, length, nil
}

func (m *Model) featureProjection(fe []float32, seqLen int) ([]float32, error) {
	channels := m.cfg.ConvDim[len(m.cfg.ConvDim)-1]
	// fe is (C, T) -> transpose to (T, C)
	projIn := make([]float32, seqLen*channels)
	for t := 0; t < seqLen; t++ {
		for c := 0; c < channels; c++ {
			projIn[t*channels+c] = fe[c*seqLen+t]
		}
	}

	if m.cfg.FeatProjLayerNorm {
		gamma, err := requireTensor(m.w, "feature_projection.layer_norm.weight")
		if err != nil {
			return nil, err
		}
		beta, err := requireTensor(m.w, "feature_projection.layer_norm.bias")
		if err != nil {
			return nil, err
		}
		for t := 0; t < seqLen; t++ {
			slice := projIn[t*channels : (t+1)*channels]
			layerNormOut(slice, slice, gamma.Data, beta.Data, m.cfg.LayerNormEps)
		}
	}

	hiddenSize := m.cfg.HiddenSize
	out := make([]float32, seqLen*hiddenSize)
	weight, err := requireTensor(m.w, "feature_projection.projection.weight")
	if err != nil {
		return nil, err
	}
	biasT, err := requireTensor(m.w, "feature_projection.projection.bias")
	if err != nil {
		return nil, err
	}
	linear2DOut(out, projIn, seqLen, channels, hiddenSize, weight, biasT.Data)
	return out, nil
}

func (m *Model) encoder(hidden []float32, seqLen int) ([]float32, int, error) {
	dim := m.cfg.HiddenSize

	posOut, err := m.posConvEmbed(hidden, seqLen, dim)
	if err != nil {
		return nil, 0, err
	}
	for i := range hidden {
		hidden[i] += posOut[i]
	}

	lnGamma, err := requireTensor(m.w, "encoder.layer_norm.weight")
	if err != nil {
		return nil, 0, err
	}
	lnBeta, err := requireTensor(m.w, "encoder.layer_norm.bias")
	if err != nil {
		return nil, 0, err
	}
	for t := 0; t < seqLen; t++ {
		slice := hidden[t*dim : (t+1)*dim]
		layerNormOut(slice, slice, lnGamma.Data, lnBeta.Data, m.cfg.LayerNormEps)
	}

	for layer := 0; layer < m.cfg.NumHiddenLayers; layer++ {
		if err := m.encoderLayer(hidden, seqLen, dim, layer); err != nil {
			return nil, 0, err
		}
	}
	return hidden, seqLen, nil
}

func (m *Model) posConvEmbed(hidden []float32, seqLen, dim int) ([]float32, error) {
	kernel := m.cfg.NumConvPosEmbeddings
	groups := m.cfg.NumConvPosEmbeddingGroups
	padding := kernel / 2
	paddedLen := seqLen + 2*padding

	// (T, C) -> (C, T) with padding
	x := make([]float32, dim*paddedLen)
	for t := 0; t < paddedLen; t++ {
		srcT := t - padding
		for c := 0; c < dim; c++ {
			var v float32
			if srcT >= 0 && srcT < seqLen {
				v = hidden[srcT*dim+c]
			}
			x[c*paddedLen+t] = v
		}
	}

	weight, err := requireTensor(m.w, "encoder.pos_conv_embed.conv.weight")
	if err != nil {
		return nil, err
	}
	biasT, err := requireTensor(m.w, "encoder.pos_conv_embed.conv.bias")
	if err != nil {
		return nil, err
	}

	convRawLen := paddedLen - kernel + 1
	convOut := make([]float32, dim*convRawLen)
	conv1DGroupedOut(convOut, x, dim, dim, paddedLen, kernel, 1, groups, weight, biasT.Data)

	geluSlice(convOut, convOut)

	// (C, T) -> (T, C)
	out := make([]float32, seqLen*dim)
	for t := 0; t < seqLen; t++ {
		for c := 0; c < dim; c++ {
			out[t*dim+c] = convOut[c*convRawLen+t]
		}
	}
	return out, nil
}

func (m *Model) encoderLayer(hidden []float32, seqLen, dim, layer int) error {
	prefix := fmt.Sprintf("encoder.layers.%d", layer)

	attnOut := make([]float32, seqLen*dim)
	if err := m.selfAttention(attnOut, hidden, seqLen, dim, prefix); err != nil {
		return err
	}

	for i := range hidden {
		hidden[i] += attnOut[i]
	}

	ln1W, err := requireTensor(m.w, prefix+".layer_norm.weight")
	if err != nil {
		return err
	}
	ln1B, err := requireTensor(m.w, prefix+".layer_norm.bias")
	if err != nil {
		return err
	}
	for t := 0; t < seqLen; t++ {
		slice := hidden[t*dim : (t+1)*dim]
		layerNormOut(slice, slice, ln1W.Data, ln1B.Data, m.cfg.LayerNormEps)
	}

	ffnOut := make([]float32, seqLen*dim)
	if err := m.feedForward(ffnOut, hidden, seqLen, dim, prefix+".feed_forward"); err != nil {
		return err
	}
	for i := range hidden {
		hidden[i] += ffnOut[i]
	}

	ln2W, err := requireTensor(m.w, prefix+".final_layer_norm.weight")
	if err != nil {
		return err
	}
	ln2B, err := requireTensor(m.w, prefix+".final_layer_norm.bias")
	if err != nil {
		return err
	}
	for t := 0; t < seqLen; t++ {
		slice := hidden[t*dim : (t+1)*dim]
		layerNormOut(slice, slice, ln2W.Data, ln2B.Data, m.cfg.LayerNormEps)
	}
	return nil
}

func (m *Model) selfAttention(out, hidden []float32, seqLen, dim int, prefix string) error {
	heads := m.cfg.NumAttentionHeads
	headDim := m.cfg.HeadDim()
	scale := float32(1.0 / math.Sqrt(float64(headDim)))

	qW, err := requireTensor(m.w, prefix+".attention.q_proj.weight")
	if err != nil {
		return err
	}
	qB, err := requireTensor(m.w, prefix+".attention.q_proj.bias")
	if err != nil {
		return err
	}
	kW, err := requireTensor(m.w, prefix+".attention.k_proj.weight")
	if err != nil {
		return err
	}
	kB, err := requireTensor(m.w, prefix+".attention.k_proj.bias")
	if err != nil {
		return err
	}
	vW, err := requireTensor(m.w, prefix+".attention.v_proj.weight")
	if err != nil {
		return err
	}
	vB, err := requireTensor(m.w, prefix+".attention.v_proj.bias")
	if err != nil {
		return err
	}
	oW, err := requireTensor(m.w, prefix+".attention.out_proj.weight")
	if err != nil {
		return err
	}
	oB, err := requireTensor(m.w, prefix+".attention.out_proj.bias")
	if err != nil {
		return err
	}

	q := make([]float32, seqLen*dim)
	k := make([]float32, seqLen*dim)
	v := make([]float32, seqLen*dim)
	linear2DOut(q, hidden, seqLen, dim, dim, qW, qB.Data)
	linear2DOut(k, hidden, seqLen, dim, dim, kW, kB.Data)
	linear2DOut(v, hidden, seqLen, dim, dim, vW, vB.Data)

	scores := make([]float32, seqLen)
	attnCtx := make([]float32, headDim)

	for t := 0; t < seqLen; t++ {
		outBase := t * dim
		for h := 0; h < heads; h++ {
			qOff := t*dim + h*headDim
			for d := 0; d < headDim; d++ {
				attnCtx[d] = 0
			}
			for j := 0; j < seqLen; j++ {
				kOff := j*dim + h*headDim
				var dot float32
				for d := 0; d < headDim; d++ {
					dot += q[qOff+d] * k[kOff+d]
				}
				scores[j] = dot * scale
			}
			softmaxRow(scores[:seqLen])

			for d := 0; d < headDim; d++ {
				var sum float32
				for j := 0; j < seqLen; j++ {
					vOff := j*dim + h*headDim
					sum += scores[j] * v[vOff+d]
				}
				attnCtx[d] = sum
			}
			copy(out[outBase+h*headDim:outBase+(h+1)*headDim], attnCtx)
		}
	}

	merged := make([]float32, seqLen*dim)
	copy(merged, out)
	linear2DOut(out, merged, seqLen, dim, dim, oW, oB.Data)
	return nil
}

func (m *Model) feedForward(out, hidden []float32, seqLen, dim int, prefix string) error {
	inter := m.cfg.IntermediateSize
	interW, err := requireTensor(m.w, prefix+".intermediate_dense.weight")
	if err != nil {
		return err
	}
	interB, err := requireTensor(m.w, prefix+".intermediate_dense.bias")
	if err != nil {
		return err
	}
	outW, err := requireTensor(m.w, prefix+".output_dense.weight")
	if err != nil {
		return err
	}
	outB, err := requireTensor(m.w, prefix+".output_dense.bias")
	if err != nil {
		return err
	}

	tmp := make([]float32, seqLen*inter)
	linear2DOut(tmp, hidden, seqLen, dim, inter, interW, interB.Data)
	geluSlice(tmp, tmp)
	linear2DOut(out, tmp, seqLen, inter, dim, outW, outB.Data)
	return nil
}
