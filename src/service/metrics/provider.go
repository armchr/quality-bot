package metrics

import (
	"context"
	"strconv"
	"sync"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/service/codeapi"
	"quality-bot/src/util"
)

// Provider provides high-level code metrics with caching.
// It abstracts away Cypher queries and provides a clean API for detectors.
type Provider struct {
	client   *codeapi.Client
	repoName string
	cfg      config.CacheConfig

	// Cached metrics
	mu               sync.RWMutex
	functionMetrics  []model.FunctionMetrics
	classMetrics     []model.ClassMetrics
	fileMetrics      []model.FileMetrics
	classPairMetrics []model.ClassPairMetrics
}

// NewProvider creates a new metrics provider
func NewProvider(client *codeapi.Client, repoName string, cfg config.CacheConfig) *Provider {
	return &Provider{
		client:   client,
		repoName: repoName,
		cfg:      cfg,
	}
}

// RepoName returns the repository name
func (p *Provider) RepoName() string {
	return p.repoName
}

// GetAllFunctionMetrics retrieves metrics for all functions
func (p *Provider) GetAllFunctionMetrics(ctx context.Context) ([]model.FunctionMetrics, error) {
	p.mu.RLock()
	if p.functionMetrics != nil {
		defer p.mu.RUnlock()
		util.Debug("Returning %d cached function metrics", len(p.functionMetrics))
		return p.functionMetrics, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if p.functionMetrics != nil {
		util.Debug("Returning %d cached function metrics (after lock upgrade)", len(p.functionMetrics))
		return p.functionMetrics, nil
	}

	util.Debug("Fetching function metrics from CodeAPI")
	metrics, err := p.fetchFunctionMetrics(ctx)
	if err != nil {
		util.Error("Failed to fetch function metrics: %v", err)
		return nil, err
	}

	util.Info("Retrieved %d function metrics", len(metrics))
	if p.cfg.Enabled {
		p.functionMetrics = metrics
		util.Debug("Function metrics cached")
	}

	return metrics, nil
}

func (p *Provider) fetchFunctionMetrics(ctx context.Context) ([]model.FunctionMetrics, error) {
	query := `
	MATCH (fs:FileScope)-[:CONTAINS*]->(f:Function)
	WHERE fs.repo = $repo_name

	OPTIONAL MATCH (c:Class)-[:CONTAINS]->(f)
	OPTIONAL MATCH (f)-[:CONTAINS*]->(cond:Conditional)
	OPTIONAL MATCH (f)-[:CONTAINS*]->(loop:Loop)
	OPTIONAL MATCH (f)-[:CONTAINS*]->(:Conditional)-[br:BRANCH]->()
	// Count nesting by finding paths and counting only Conditional/Loop nodes in them
	OPTIONAL MATCH path = (f)-[:CONTAINS*]->(deepest)
	WHERE deepest:Conditional OR deepest:Loop
	OPTIONAL MATCH (caller:Function)-[:CALLS]->(f)
	OPTIONAL MATCH (f)-[:CALLS]->(callee:Function)
	OPTIONAL MATCH (f)-[:CALLS]->(ext:Function)<-[:CONTAINS]-(other:Class)
	WHERE other <> c
	OPTIONAL MATCH (f)-[:USES]->(own_field:Field)<-[:CONTAINS]-(c)
	OPTIONAL MATCH (f)-[:USES]->(ext_field:Field)<-[:CONTAINS]-(ext_class:Class)
	WHERE ext_class <> c

	WITH fs, f, c,
	     count(DISTINCT cond) as conditional_count,
	     count(DISTINCT loop) as loop_count,
	     count(DISTINCT br) as branch_count,
	     max(size([n IN nodes(path) WHERE n:Conditional OR n:Loop])) as max_nesting_depth,
	     count(DISTINCT caller) as caller_count,
	     count(DISTINCT callee) as callee_count,
	     count(DISTINCT other) as external_calls,
	     count(DISTINCT own_field) as own_field_uses,
	     count(DISTINCT ext_field) as external_field_uses

	RETURN
	    f.id as id,
	    f.name as name,
	    fs.path as file_path,
	    f.range as range,
	    c.name as class_name,
	    COALESCE(f.param_count, 0) as parameter_count,
	    (1 + loop_count + branch_count) as cyclomatic_complexity,
	    conditional_count,
	    loop_count,
	    branch_count,
	    COALESCE(max_nesting_depth, 0) as max_nesting_depth,
	    caller_count,
	    callee_count,
	    external_calls,
	    own_field_uses,
	    external_field_uses
	`

	results, err := p.client.ExecuteCypher(ctx, p.repoName, query)
	if err != nil {
		return nil, err
	}

	metrics := make([]model.FunctionMetrics, 0, len(results))
	for _, r := range results {
		startLine, endLine := parseRange(getString(r, "range"))
		lineCount := endLine - startLine
		if lineCount < 0 {
			lineCount = 0
		}

		metrics = append(metrics, model.FunctionMetrics{
			ID:                   getString(r, "id"),
			Name:                 getString(r, "name"),
			FilePath:             getString(r, "file_path"),
			StartLine:            startLine,
			EndLine:              endLine,
			ClassName:            getString(r, "class_name"),
			LineCount:            lineCount,
			ParameterCount:       getInt(r, "parameter_count"),
			CyclomaticComplexity: getInt(r, "cyclomatic_complexity"),
			ConditionalCount:     getInt(r, "conditional_count"),
			LoopCount:            getInt(r, "loop_count"),
			BranchCount:          getInt(r, "branch_count"),
			MaxNestingDepth:      getInt(r, "max_nesting_depth"),
			CallerCount:          getInt(r, "caller_count"),
			CalleeCount:          getInt(r, "callee_count"),
			ExternalCalls:        getInt(r, "external_calls"),
			OwnFieldUses:         getInt(r, "own_field_uses"),
			ExternalFieldUses:    getInt(r, "external_field_uses"),
		})
	}

	return metrics, nil
}

// GetAllClassMetrics retrieves metrics for all classes
func (p *Provider) GetAllClassMetrics(ctx context.Context) ([]model.ClassMetrics, error) {
	p.mu.RLock()
	if p.classMetrics != nil {
		defer p.mu.RUnlock()
		util.Debug("Returning %d cached class metrics", len(p.classMetrics))
		return p.classMetrics, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.classMetrics != nil {
		util.Debug("Returning %d cached class metrics (after lock upgrade)", len(p.classMetrics))
		return p.classMetrics, nil
	}

	util.Debug("Fetching class metrics from CodeAPI")
	metrics, err := p.fetchClassMetrics(ctx)
	if err != nil {
		util.Error("Failed to fetch class metrics: %v", err)
		return nil, err
	}

	util.Info("Retrieved %d class metrics", len(metrics))
	if p.cfg.Enabled {
		p.classMetrics = metrics
		util.Debug("Class metrics cached")
	}

	return metrics, nil
}

func (p *Provider) fetchClassMetrics(ctx context.Context) ([]model.ClassMetrics, error) {
	query := `
	MATCH (fs:FileScope)-[:CONTAINS]->(c:Class)
	WHERE fs.repo = $repo_name

	OPTIONAL MATCH (c)-[:CONTAINS]->(m:Function)
	OPTIONAL MATCH (c)-[:CONTAINS]->(f:Field)
	OPTIONAL MATCH (c)-[:CONTAINS]->(pf:Field)
	WHERE pf.type IN ['string', 'int', 'float', 'bool', 'int64', 'float64', 'String', 'Integer', 'Boolean', 'Double']
	OPTIONAL MATCH (c)-[:CONTAINS]->(:Function)-[:CALLS]->(:Function)<-[:CONTAINS]-(dep:Class)
	WHERE dep <> c
	OPTIONAL MATCH (other:Class)-[:CONTAINS]->(:Function)-[:CALLS]->(:Function)<-[:CONTAINS]-(c)
	WHERE other <> c
	OPTIONAL MATCH inheritance_path = (c)-[:INHERITS_FROM*]->(parent:Class)

	WITH fs, c,
	     count(DISTINCT m) as method_count,
	     count(DISTINCT f) as field_count,
	     count(DISTINCT pf) as primitive_field_count,
	     count(DISTINCT dep) as dependency_count,
	     count(DISTINCT other) as dependent_count,
	     max(length(inheritance_path)) as inheritance_depth

	RETURN
	    c.id as id,
	    c.name as name,
	    fs.path as file_path,
	    c.range as range,
	    method_count,
	    field_count,
	    primitive_field_count,
	    dependency_count,
	    dependent_count,
	    COALESCE(inheritance_depth, 0) as inheritance_depth
	`

	results, err := p.client.ExecuteCypher(ctx, p.repoName, query)
	if err != nil {
		return nil, err
	}

	metrics := make([]model.ClassMetrics, 0, len(results))
	for _, r := range results {
		startLine, endLine := parseRange(getString(r, "range"))
		lineCount := endLine - startLine
		if lineCount < 0 {
			lineCount = 0
		}

		metrics = append(metrics, model.ClassMetrics{
			ID:                  getString(r, "id"),
			Name:                getString(r, "name"),
			FilePath:            getString(r, "file_path"),
			StartLine:           startLine,
			EndLine:             endLine,
			LineCount:           lineCount,
			MethodCount:         getInt(r, "method_count"),
			FieldCount:          getInt(r, "field_count"),
			PrimitiveFieldCount: getInt(r, "primitive_field_count"),
			DependencyCount:     getInt(r, "dependency_count"),
			DependentCount:      getInt(r, "dependent_count"),
			InheritanceDepth:    getInt(r, "inheritance_depth"),
		})
	}

	return metrics, nil
}

// GetAllFileMetrics retrieves metrics for all files
func (p *Provider) GetAllFileMetrics(ctx context.Context) ([]model.FileMetrics, error) {
	p.mu.RLock()
	if p.fileMetrics != nil {
		defer p.mu.RUnlock()
		util.Debug("Returning %d cached file metrics", len(p.fileMetrics))
		return p.fileMetrics, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.fileMetrics != nil {
		util.Debug("Returning %d cached file metrics (after lock upgrade)", len(p.fileMetrics))
		return p.fileMetrics, nil
	}

	util.Debug("Fetching file metrics from CodeAPI")
	metrics, err := p.fetchFileMetrics(ctx)
	if err != nil {
		util.Error("Failed to fetch file metrics: %v", err)
		return nil, err
	}

	util.Info("Retrieved %d file metrics", len(metrics))
	if p.cfg.Enabled {
		p.fileMetrics = metrics
		util.Debug("File metrics cached")
	}

	return metrics, nil
}

func (p *Provider) fetchFileMetrics(ctx context.Context) ([]model.FileMetrics, error) {
	query := `
	MATCH (fs:FileScope)
	WHERE fs.repo = $repo_name

	OPTIONAL MATCH (fs)-[:CONTAINS]->(f:Function)
	OPTIONAL MATCH (fs)-[:CONTAINS]->(c:Class)

	WITH fs,
	     count(DISTINCT f) as function_count,
	     count(DISTINCT c) as class_count,
	     collect(DISTINCT f) as functions

	// Calculate complexity for each function
	UNWIND CASE WHEN size(functions) > 0 THEN functions ELSE [null] END as func
	OPTIONAL MATCH (func)-[:CONTAINS*]->(loop:Loop)
	OPTIONAL MATCH (func)-[:CONTAINS*]->(:Conditional)-[br:BRANCH]->()

	WITH fs, function_count, class_count, func,
	     CASE WHEN func IS NOT NULL THEN 1 + count(DISTINCT loop) + count(DISTINCT br) ELSE 0 END as cc

	WITH fs, function_count, class_count,
	     sum(cc) as total_cyclomatic_complexity,
	     max(cc) as max_function_complexity

	RETURN
	    fs.path as path,
	    fs.language as language,
	    fs.range as range,
	    function_count,
	    class_count,
	    total_cyclomatic_complexity,
	    max_function_complexity
	`

	results, err := p.client.ExecuteCypher(ctx, p.repoName, query)
	if err != nil {
		return nil, err
	}

	metrics := make([]model.FileMetrics, 0, len(results))
	for _, r := range results {
		funcCount := getInt(r, "function_count")
		totalCC := getInt(r, "total_cyclomatic_complexity")
		avgCC := 0.0
		if funcCount > 0 {
			avgCC = float64(totalCC) / float64(funcCount)
		}

		// Parse range to get line count - format is (0,0)-(lineCount,0)
		_, endLine := parseRange(getString(r, "range"))

		metrics = append(metrics, model.FileMetrics{
			Path:                      getString(r, "path"),
			Language:                  getString(r, "language"),
			LineCount:                 endLine,
			FunctionCount:             funcCount,
			ClassCount:                getInt(r, "class_count"),
			TotalCyclomaticComplexity: totalCC,
			MaxFunctionComplexity:     getInt(r, "max_function_complexity"),
			AvgFunctionComplexity:     avgCC,
		})
	}

	return metrics, nil
}

// GetClassPairMetrics retrieves coupling metrics between class pairs
func (p *Provider) GetClassPairMetrics(ctx context.Context) ([]model.ClassPairMetrics, error) {
	p.mu.RLock()
	if p.classPairMetrics != nil {
		defer p.mu.RUnlock()
		util.Debug("Returning %d cached class pair metrics", len(p.classPairMetrics))
		return p.classPairMetrics, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.classPairMetrics != nil {
		util.Debug("Returning %d cached class pair metrics (after lock upgrade)", len(p.classPairMetrics))
		return p.classPairMetrics, nil
	}

	util.Debug("Fetching class pair metrics from CodeAPI")
	metrics, err := p.fetchClassPairMetrics(ctx)
	if err != nil {
		util.Error("Failed to fetch class pair metrics: %v", err)
		return nil, err
	}

	util.Info("Retrieved %d class pair metrics", len(metrics))
	if p.cfg.Enabled {
		p.classPairMetrics = metrics
		util.Debug("Class pair metrics cached")
	}

	return metrics, nil
}

func (p *Provider) fetchClassPairMetrics(ctx context.Context) ([]model.ClassPairMetrics, error) {
	query := `
	MATCH (fs1:FileScope)-[:CONTAINS]->(c1:Class)-[:CONTAINS]->(f1:Function)-[:CALLS]->(f2:Function)<-[:CONTAINS]-(c2:Class)<-[:CONTAINS]-(fs2:FileScope)
	WHERE c1 <> c2 AND fs1.repo = $repo_name AND fs2.repo = $repo_name

	WITH c1, c2, count(*) as calls_1_to_2

	OPTIONAL MATCH (c2)-[:CONTAINS]->(f3:Function)-[:CALLS]->(f4:Function)<-[:CONTAINS]-(c1)

	WITH c1, c2, calls_1_to_2, count(f3) as calls_2_to_1

	OPTIONAL MATCH (c1)-[:CONTAINS]->(:Function)-[:USES]->(field:Field)<-[:CONTAINS]-(c2)
	OPTIONAL MATCH (c2)-[:CONTAINS]->(:Function)-[:USES]->(field2:Field)<-[:CONTAINS]-(c1)

	WITH c1, c2, calls_1_to_2, calls_2_to_1,
	     count(DISTINCT field) + count(DISTINCT field2) as shared_field_access

	WHERE calls_1_to_2 > 0 OR calls_2_to_1 > 0

	RETURN
	    c1.name as class1_name,
	    c1.file_path as class1_file,
	    c2.name as class2_name,
	    c2.file_path as class2_file,
	    calls_1_to_2,
	    calls_2_to_1,
	    shared_field_access
	`

	results, err := p.client.ExecuteCypher(ctx, p.repoName, query)
	if err != nil {
		return nil, err
	}

	metrics := make([]model.ClassPairMetrics, 0, len(results))
	for _, r := range results {
		metrics = append(metrics, model.ClassPairMetrics{
			Class1Name:        getString(r, "class1_name"),
			Class1File:        getString(r, "class1_file"),
			Class2Name:        getString(r, "class2_name"),
			Class2File:        getString(r, "class2_file"),
			Calls1To2:         getInt(r, "calls_1_to_2"),
			Calls2To1:         getInt(r, "calls_2_to_1"),
			SharedFieldAccess: getInt(r, "shared_field_access"),
		})
	}

	return metrics, nil
}

// ClearCache clears all cached metrics
func (p *Provider) ClearCache() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.functionMetrics = nil
	p.classMetrics = nil
	p.fileMetrics = nil
	p.classPairMetrics = nil
	util.Debug("Metrics cache cleared")
}

// Helper functions
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func getFloat(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// parseRange parses a range string in format "(startLine,startCol)-(endLine,endCol)"
// and returns startLine and endLine
func parseRange(rangeStr string) (startLine, endLine int) {
	if rangeStr == "" {
		return 0, 0
	}

	// Format: (62,4)-(75,5)
	// Extract numbers using simple parsing
	var nums []int
	var current string
	for _, ch := range rangeStr {
		if ch >= '0' && ch <= '9' {
			current += string(ch)
		} else if current != "" {
			if n, err := strconv.Atoi(current); err == nil {
				nums = append(nums, n)
			}
			current = ""
		}
	}
	if current != "" {
		if n, err := strconv.Atoi(current); err == nil {
			nums = append(nums, n)
		}
	}

	// nums should be [startLine, startCol, endLine, endCol]
	if len(nums) >= 4 {
		return nums[0], nums[2]
	} else if len(nums) >= 2 {
		return nums[0], nums[1]
	}
	return 0, 0
}
