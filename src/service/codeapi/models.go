package codeapi

// CypherRequest represents a request to execute a Cypher query
type CypherRequest struct {
	RepoName   string         `json:"repo_name"`
	Query      string         `json:"query"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// CypherResponse represents the response from a Cypher query
type CypherResponse struct {
	Results []map[string]any `json:"results"`
}

// SimilarCodeRequest represents a request to search for similar code
type SimilarCodeRequest struct {
	RepoName    string `json:"repo_name"`
	CodeSnippet string `json:"code_snippet"`
	Language    string `json:"language"`
	Limit       int    `json:"limit,omitempty"`
	IncludeCode bool   `json:"include_code,omitempty"`
}

// SimilarCodeResponse represents the response from a similarity search
type SimilarCodeResponse struct {
	Results []SimilarCodeResult `json:"results"`
	Success bool                `json:"success"`
}

// SimilarCodeResult represents a single result from similarity search
type SimilarCodeResult struct {
	Chunk SimilarCodeChunk `json:"chunk"`
	Score float64          `json:"score"`
	Code  string           `json:"code,omitempty"`
}

// SimilarCodeChunk represents the matched code chunk metadata
type SimilarCodeChunk struct {
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	ChunkType string `json:"chunk_type"`
	Name      string `json:"name"`
}

// FunctionsRequest represents a request to get functions
type FunctionsRequest struct {
	RepoName string `json:"repo_name"`
	FilePath string `json:"file_path,omitempty"`
}

// FunctionsResponse represents the response with functions
type FunctionsResponse struct {
	Functions []FunctionInfo `json:"functions"`
}

// FunctionInfo contains basic function information
type FunctionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	ClassName string `json:"class_name,omitempty"`
}

// SnippetRequest represents a request to get a code snippet
type SnippetRequest struct {
	RepoName  string `json:"repo_name"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// SnippetResponse represents the response with a code snippet
type SnippetResponse struct {
	RepoName   string `json:"repo_name"`
	FilePath   string `json:"file_path"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
	Code       string `json:"code"`
	TotalLines int    `json:"total_lines"`
}
