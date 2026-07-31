package neural

import "encoding/json"

// Config mirrors m-a-p/MERT-v1-95M config.json fields used at inference.
type Config struct {
	HiddenSize                 int     `json:"hidden_size"`
	NumHiddenLayers            int     `json:"num_hidden_layers"`
	NumAttentionHeads          int     `json:"num_attention_heads"`
	IntermediateSize           int     `json:"intermediate_size"`
	HiddenDropout              float64 `json:"hidden_dropout"`
	AttentionDropout           float64 `json:"attention_dropout"`
	LayerNormEps               float64 `json:"layer_norm_eps"`
	FeatExtractNorm            string  `json:"feat_extract_norm"`
	FeatExtractActivation      string  `json:"feat_extract_activation"`
	ConvDim                    []int   `json:"conv_dim"`
	ConvStride                 []int   `json:"conv_stride"`
	ConvKernel                 []int   `json:"conv_kernel"`
	ConvBias                   bool    `json:"conv_bias"`
	NumConvPosEmbeddings       int     `json:"num_conv_pos_embeddings"`
	NumConvPosEmbeddingGroups  int     `json:"num_conv_pos_embedding_groups"`
	FeatProjLayerNorm          bool    `json:"feat_proj_layer_norm"`
	FeatProjDropout            float64 `json:"feat_proj_dropout"`
	SampleRate                 int     `json:"sample_rate"`
	FeatureExtractorCQT        bool    `json:"feature_extractor_cqt"`
}

func parseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 24000
	}
	if cfg.LayerNormEps == 0 {
		cfg.LayerNormEps = 1e-5
	}
	return cfg, nil
}

func (c Config) HeadDim() int {
	return c.HiddenSize / c.NumAttentionHeads
}

func (c Config) InputsToLogitsRatio() int {
	r := 1
	for _, s := range c.ConvStride {
		r *= s
	}
	return r
}
