package download

type Model struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	HFRepo   string  `json:"hfRepo"`
	HFFile   string  `json:"hfFile"`
	SizeGB   float64 `json:"sizeGB"`
	MinRAMGB int     `json:"minRamGB"`
	Quality  string  `json:"quality"`
	Tagline  string  `json:"tagline"`
}

var CuratedModels = []Model{
	{
		ID:       "phi-4-mini",
		Name:     "Phi-4-mini 3.8B",
		HFRepo:   "microsoft/Phi-4-mini-instruct-gguf",
		HFFile:   "Phi-4-mini-instruct-q4_k_m.gguf",
		SizeGB:   2.5,
		MinRAMGB: 4,
		Quality:  "★★★★",
		Tagline:  "Best reasoning — Microsoft's compact powerhouse",
	},
	{
		ID:       "qwen3-3b",
		Name:     "Qwen3 3B",
		HFRepo:   "Qwen/Qwen3-3B-Instruct-GGUF",
		HFFile:   "qwen3-3b-instruct-q4_k_m.gguf",
		SizeGB:   2.0,
		MinRAMGB: 4,
		Quality:  "★★★★",
		Tagline:  "Best coding — Alibaba's strong reasoning model",
	},
	{
		ID:       "llama-3.2-3b",
		Name:     "Llama 3.2 3B",
		HFRepo:   "unsloth/Llama-3.2-3B-Instruct-GGUF",
		HFFile:   "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		SizeGB:   2.5,
		MinRAMGB: 4,
		Quality:  "★★★",
		Tagline:  "Solid general purpose — Meta's reliable workhorse",
	},
	{
		ID:       "gemma-3-1b",
		Name:     "Gemma 3 1B",
		HFRepo:   "ggml-org/gemma-3-1b-it-GGUF",
		HFFile:   "gemma-3-1b-it-Q4_K_M.gguf",
		SizeGB:   0.7,
		MinRAMGB: 2,
		Quality:  "★★",
		Tagline:  "Fastest — Google's tiny model, runs on anything",
	},
	{
		ID:       "qwen3-1.5b",
		Name:     "Qwen3 1.5B",
		HFRepo:   "Qwen/Qwen3-1.5B-Instruct-GGUF",
		HFFile:   "qwen3-1.5b-instruct-q4_k_m.gguf",
		SizeGB:   1.0,
		MinRAMGB: 2,
		Quality:  "★★",
		Tagline:  "Light coding — when you need speed over depth",
	},
}

func Recommend(freeRAMGB int) *Model {
	if freeRAMGB < 1 {
		freeRAMGB = 4
	}
	best := 0
	for i, m := range CuratedModels {
		if freeRAMGB < m.MinRAMGB {
			continue
		}
		if best == 0 || (m.SizeGB > CuratedModels[best].SizeGB && m.MinRAMGB <= freeRAMGB) {
			best = i
		}
	}
	if best == 0 && len(CuratedModels) > 0 {
		best = len(CuratedModels) - 1
	}
	return &CuratedModels[best]
}
