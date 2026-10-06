package embedding

const OnnxRuntimeVersion = "1.30.0"

const onnxRuntimeReleaseURL = "https://github.com/microsoft/onnxruntime/releases/download/v" + OnnxRuntimeVersion

// CatalogFileURL points to one downloadable file of a model preset.
type CatalogFileURL struct {
	URL       string `json:"url"`
	DestFile  string `json:"dest_file"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256,omitempty"`
}

// CatalogModel is a downloadable embedding model preset backed by HuggingFace files.
type CatalogModel struct {
	PresetID        string           `json:"preset_id"`
	DisplayName     string           `json:"display_name"`
	Dim             int              `json:"dim"`
	Description     string           `json:"description"`
	FileURLs        []CatalogFileURL `json:"file_urls"`
	ApproxSizeBytes int64            `json:"approx_size_bytes"`
	Manifest        Manifest         `json:"-"`
}

// CatalogRuntime is a downloadable onnxruntime shared library for one platform.
type CatalogRuntime struct {
	PresetID        string `json:"preset_id"`
	GOOS            string `json:"goos"`
	GOArch          string `json:"goarch"`
	DisplayName     string `json:"display_name"`
	URL             string `json:"url"`
	Archive         string `json:"archive"`
	ArchiveSHA256   string `json:"archive_sha256,omitempty"`
	InnerPath       string `json:"inner_path"`
	SHA256          string `json:"sha256,omitempty"`
	SizeBytes       int64  `json:"size_bytes"`
	ApproxSizeBytes int64  `json:"approx_size_bytes"`
}

// Catalog is the static list of downloadable models and runtime libraries.
type Catalog struct {
	Models   []CatalogModel   `json:"models"`
	Runtimes []CatalogRuntime `json:"runtimes"`
}

// BuiltinCatalog returns the static download catalog shipped with the server.
func BuiltinCatalog() Catalog {
	return Catalog{
		Models: []CatalogModel{
			{
				PresetID:    "all-minilm-l6-v2",
				DisplayName: "all-MiniLM-L6-v2",
				Dim:         384,
				Description: "Fast English sentence embedding model, 384 dimensions, WordPiece tokenizer, mean pooling.",
				FileURLs: []CatalogFileURL{
					{
						URL:       "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx",
						DestFile:  ModelFileName,
						SizeBytes: 90405214,
						SHA256:    "6fd5d72fe4589f189f8ebc006442dbb529bb7ce38f8082112682524616046452",
					},
					{
						URL:       "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/vocab.txt",
						DestFile:  VocabFileName,
						SizeBytes: 231508,
						SHA256:    "07eced375cec144d27c900241f3e339478dec958f92fddbc551f295c992038a3",
					},
				},
				ApproxSizeBytes: 90636722,
				Manifest: Manifest{
					SchemaVersion: ManifestSchemaVersion,
					ModelID:       "all-minilm-l6-v2",
					DisplayName:   "all-MiniLM-L6-v2",
					Dim:           384,
					MaxSeqTokens:  256,
					Pooling:       PoolingMean,
					Normalize:     true,
					QueryPrefix:   "",
					Inputs:        []string{InputIDsName, AttentionMaskName, TokenTypeIDsName},
					Outputs:       []string{"last_hidden_state"},
					Tokenizer:     TokenizerWordPiece,
					Lowercase:     true,
					StripAccents:  true,
				},
			},
			{
				PresetID:    "distiluse-multilingual-cased-v2",
				DisplayName: "distiluse-base-multilingual-cased-v2",
				Dim:         768,
				Description: "Multilingual distilBERT sentence embedding model covering Vietnamese and 50+ languages, 768 dimensions, WordPiece tokenizer, mean pooling.",
				FileURLs: []CatalogFileURL{
					{
						URL:       "https://huggingface.co/sentence-transformers/distiluse-base-multilingual-cased-v2/resolve/main/onnx/model.onnx",
						DestFile:  ModelFileName,
						SizeBytes: 539049415,
						SHA256:    "698b5efe2a2ec26e6ec492906c0efa168659994e9838a131f88994fb6f637520",
					},
					{
						URL:       "https://huggingface.co/sentence-transformers/distiluse-base-multilingual-cased-v2/resolve/main/vocab.txt",
						DestFile:  VocabFileName,
						SizeBytes: 995526,
						SHA256:    "fe0fda7c425b48c516fc8f160d594c8022a0808447475c1a7c6d6479763f310c",
					},
				},
				ApproxSizeBytes: 540044941,
				Manifest: Manifest{
					SchemaVersion: ManifestSchemaVersion,
					ModelID:       "distiluse-multilingual-cased-v2",
					DisplayName:   "distiluse-base-multilingual-cased-v2",
					Dim:           768,
					MaxSeqTokens:  128,
					Pooling:       PoolingMean,
					Normalize:     true,
					QueryPrefix:   "",
					Inputs:        []string{InputIDsName, AttentionMaskName},
					Outputs:       []string{"last_hidden_state"},
					Tokenizer:     TokenizerWordPiece,
					Lowercase:     false,
					StripAccents:  false,
				},
			},
			{
				PresetID:    "bge-m3",
				DisplayName: "BAAI BGE-M3 (Quantized INT8, 1024-dim)",
				Dim:         1024,
				Description: "State-of-the-art multilingual embedding model covering Vietnamese, 1024 dimensions, INT8 quantized ONNX, CLS pooling.",
				FileURLs: []CatalogFileURL{
					{
						URL:       "https://huggingface.co/Xenova/bge-m3/resolve/main/onnx/model_int8.onnx",
						DestFile:  ModelFileName,
						SizeBytes: 568453472,
					},
					{
						URL:       "https://huggingface.co/Xenova/bge-m3/resolve/main/tokenizer.json",
						DestFile:  TokenizerFileName,
						SizeBytes: 17148906,
					},
				},
				ApproxSizeBytes: 585602378,
				Manifest: Manifest{
					SchemaVersion: ManifestSchemaVersion,
					ModelID:       "bge-m3",
					DisplayName:   "BAAI BGE-M3 (Quantized INT8, 1024-dim)",
					Dim:           1024,
					MaxSeqTokens:  512,
					Pooling:       PoolingCLS,
					Normalize:     true,
					QueryPrefix:   "",
					Inputs:        []string{InputIDsName, AttentionMaskName},
					Outputs:       []string{"last_hidden_state"},
					Tokenizer:     TokenizerUnigram,
					Lowercase:     false,
					StripAccents:  false,
				},
			},
		},
		Runtimes: []CatalogRuntime{
			{
				PresetID:        "onnxruntime-windows-amd64",
				GOOS:            "windows",
				GOArch:          "amd64",
				DisplayName:     "ONNX Runtime " + OnnxRuntimeVersion + " (Windows x64)",
				URL:             onnxRuntimeReleaseURL + "/onnxruntime-win-x64-" + OnnxRuntimeVersion + ".zip",
				Archive:         "zip",
				ArchiveSHA256:   "c6ba983baf5681af108599675d2a89c2d145512d02de28aed0bff177cd0ba949",
				InnerPath:       "onnxruntime-win-x64-" + OnnxRuntimeVersion + "/lib/onnxruntime.dll",
				SizeBytes:       82645522,
				ApproxSizeBytes: 16462648,
			},
			{
				PresetID:        "onnxruntime-linux-amd64",
				GOOS:            "linux",
				GOArch:          "amd64",
				DisplayName:     "ONNX Runtime " + OnnxRuntimeVersion + " (Linux x64)",
				URL:             onnxRuntimeReleaseURL + "/onnxruntime-linux-x64-" + OnnxRuntimeVersion + ".tgz",
				Archive:         "tgz",
				ArchiveSHA256:   "a5ed5a3cac51fbb2e90da632ae43d19212faaa20e76484e62bcb7c23ddb3b3fd",
				InnerPath:       "onnxruntime-linux-x64-" + OnnxRuntimeVersion + "/lib/libonnxruntime.so." + OnnxRuntimeVersion,
				SHA256:          "8902e52a04bf6bf26f1613746811d6194f9c97ce7a6a7a28452b48522d3bd8d0",
				SizeBytes:       11306877,
				ApproxSizeBytes: 28985152,
			},
			{
				PresetID:        "onnxruntime-linux-arm64",
				GOOS:            "linux",
				GOArch:          "arm64",
				DisplayName:     "ONNX Runtime " + OnnxRuntimeVersion + " (Linux ARM64)",
				URL:             onnxRuntimeReleaseURL + "/onnxruntime-linux-aarch64-" + OnnxRuntimeVersion + ".tgz",
				Archive:         "tgz",
				ArchiveSHA256:   "e16a27a8ed330bbc698df7330b0cf56e722f354e3bcc92118682c74ef3c3e3da",
				InnerPath:       "onnxruntime-linux-aarch64-" + OnnxRuntimeVersion + "/lib/libonnxruntime.so." + OnnxRuntimeVersion,
				SizeBytes:       10269495,
				ApproxSizeBytes: 28985152,
			},
			{
				PresetID:        "onnxruntime-darwin-arm64",
				GOOS:            "darwin",
				GOArch:          "arm64",
				DisplayName:     "ONNX Runtime " + OnnxRuntimeVersion + " (macOS Apple Silicon)",
				URL:             onnxRuntimeReleaseURL + "/onnxruntime-osx-arm64-" + OnnxRuntimeVersion + ".tgz",
				Archive:         "tgz",
				ArchiveSHA256:   "6ebb5062a934537c352937821f9fe9718e7de1a2db1122a93dd363ffd53a7012",
				InnerPath:       "onnxruntime-osx-arm64-" + OnnxRuntimeVersion + "/lib/libonnxruntime." + OnnxRuntimeVersion + ".dylib",
				SizeBytes:       42373116,
				ApproxSizeBytes: 43879424,
			},
		},
	}
}
