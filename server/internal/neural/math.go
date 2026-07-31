package neural

import "math"

func gelu(x float32) float32 {
	// Exact GELU used by PyTorch / transformers.
	const invSqrt2 = 0.7071067811865476
	xf := float64(x)
	return float32(xf * 0.5 * (1.0 + math.Erf(xf*invSqrt2)))
}

func geluSlice(dst, src []float32) {
	for i := range src {
		dst[i] = gelu(src[i])
	}
}

func softmaxRow(scores []float32) {
	if len(scores) == 0 {
		return
	}
	max := scores[0]
	for _, v := range scores[1:] {
		if v > max {
			max = v
		}
	}
	var sum float64
	for i, v := range scores {
		e := math.Exp(float64(v - max))
		scores[i] = float32(e)
		sum += e
	}
	inv := float32(1.0 / sum)
	for i := range scores {
		scores[i] *= inv
	}
}

func groupNormChannels(out []float32, x []float32, channels, length int, gamma, beta []float32, eps float64) {
	for c := 0; c < channels; c++ {
		base := c * length
		slice := x[base : base+length]
		outSlice := out[base : base+length]
		layerNormOut(outSlice, slice, []float32{gamma[c]}, []float32{beta[c]}, eps)
	}
}

func layerNormOut(out, x, gamma, beta []float32, eps float64) {
	var mean, varSum float64
	for _, v := range x {
		mean += float64(v)
	}
	mean /= float64(len(x))
	for _, v := range x {
		d := float64(v) - mean
		varSum += d * d
	}
	variance := varSum / float64(len(x))
	invStd := float32(1.0 / math.Sqrt(variance+eps))
	for i, v := range x {
		out[i] = (v - float32(mean)) * invStd
		if gamma != nil {
			g := gamma[0]
			if len(gamma) == len(x) {
				g = gamma[i]
			}
			out[i] *= g
		}
		if beta != nil {
			b := beta[0]
			if len(beta) == len(x) {
				b = beta[i]
			}
			out[i] += b
		}
	}
}

func linearOut(out []float32, x []float32, weight *Tensor, bias []float32) {
	// weight: [out, in], x: [in], out: [outFeatures]
	outFeatures := weight.Dims[0]
	inFeatures := weight.Dims[1]
	for o := 0; o < outFeatures; o++ {
		var sum float32
		wBase := o * inFeatures
		for i := 0; i < inFeatures; i++ {
			sum += weight.Data[wBase+i] * x[i]
		}
		if bias != nil {
			sum += bias[o]
		}
		out[o] = sum
	}
}

func linear2DOut(out []float32, x []float32, seqLen, inFeatures, outFeatures int, weight *Tensor, bias []float32) {
	for t := 0; t < seqLen; t++ {
		xOff := t * inFeatures
		oOff := t * outFeatures
		linearOut(out[oOff:oOff+outFeatures], x[xOff:xOff+inFeatures], weight, bias)
	}
}

func conv1DOut(
	out []float32,
	x []float32,
	inChannels, outChannels, inLen, kernel, stride int,
	weight *Tensor,
) {
	outLen := (inLen-kernel)/stride + 1
	if outLen < 0 {
		outLen = 0
	}
	for oc := 0; oc < outChannels; oc++ {
		for ot := 0; ot < outLen; ot++ {
			var sum float32
			inT0 := ot * stride
			for ic := 0; ic < inChannels; ic++ {
				for k := 0; k < kernel; k++ {
					it := inT0 + k
					if it >= 0 && it < inLen {
						wIdx := oc*inChannels*kernel + ic*kernel + k
						xIdx := ic*inLen + it
						sum += weight.Data[wIdx] * x[xIdx]
					}
				}
			}
			out[oc*outLen+ot] = sum
		}
	}
}

func conv1DGroupedOut(
	out []float32,
	x []float32,
	inChannels, outChannels, inLen, kernel, stride, groups int,
	weight *Tensor,
	bias []float32,
) {
	outLen := (inLen-kernel)/stride + 1
	if outLen < 0 {
		outLen = 0
	}
	chPerGroup := inChannels / groups
	outPerGroup := outChannels / groups
	for oc := 0; oc < outChannels; oc++ {
		g := oc / outPerGroup
		inGroupStart := g * chPerGroup
		for ot := 0; ot < outLen; ot++ {
			var sum float32
			inT0 := ot * stride
			for icLocal := 0; icLocal < chPerGroup; icLocal++ {
				ic := inGroupStart + icLocal
				for k := 0; k < kernel; k++ {
					it := inT0 + k
					if it >= 0 && it < inLen {
						wIdx := oc*chPerGroup*kernel + icLocal*kernel + k
						xIdx := ic*inLen + it
						sum += weight.Data[wIdx] * x[xIdx]
					}
				}
			}
			if bias != nil {
				sum += bias[oc]
			}
			out[oc*outLen+ot] = sum
		}
	}
}

func meanPoolTime(out []float32, hidden []float32, seqLen, dim int) {
	for d := 0; d < dim; d++ {
		var sum float64
		for t := 0; t < seqLen; t++ {
			sum += float64(hidden[t*dim+d])
		}
		out[d] = float32(sum / float64(seqLen))
	}
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}
