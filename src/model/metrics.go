package model

// FunctionMetrics contains metrics for a single function/method
type FunctionMetrics struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	ClassName string `json:"class_name,omitempty"`

	// Size metrics
	LineCount      int `json:"line_count"`
	ParameterCount int `json:"parameter_count"`

	// Complexity metrics
	CyclomaticComplexity int `json:"cyclomatic_complexity"`
	ConditionalCount     int `json:"conditional_count"`
	LoopCount            int `json:"loop_count"`
	BranchCount          int `json:"branch_count"`
	MaxNestingDepth      int `json:"max_nesting_depth"`

	// Coupling metrics
	CallerCount       int `json:"caller_count"`
	CalleeCount       int `json:"callee_count"`
	ExternalCalls     int `json:"external_calls"`
	OwnFieldUses      int `json:"own_field_uses"`
	ExternalFieldUses int `json:"external_field_uses"`
}

// ClassMetrics contains metrics for a single class
type ClassMetrics struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`

	// Size metrics
	LineCount   int `json:"line_count"`
	MethodCount int `json:"method_count"`
	FieldCount  int `json:"field_count"`

	// Composition metrics
	PrimitiveFieldCount int `json:"primitive_field_count"`

	// Coupling metrics
	DependencyCount  int `json:"dependency_count"`
	DependentCount   int `json:"dependent_count"`
	InheritanceDepth int `json:"inheritance_depth"`
}

// FileMetrics contains metrics for a single file
type FileMetrics struct {
	Path     string `json:"path"`
	Language string `json:"language"`

	// Size metrics
	LineCount     int `json:"line_count"`
	FunctionCount int `json:"function_count"`
	ClassCount    int `json:"class_count"`

	// Aggregated complexity
	TotalCyclomaticComplexity int     `json:"total_cyclomatic_complexity"`
	MaxFunctionComplexity     int     `json:"max_function_complexity"`
	AvgFunctionComplexity     float64 `json:"avg_function_complexity"`
}

// ClassPairMetrics contains coupling metrics between two classes
type ClassPairMetrics struct {
	Class1Name        string `json:"class1_name"`
	Class1File        string `json:"class1_file"`
	Class2Name        string `json:"class2_name"`
	Class2File        string `json:"class2_file"`
	Calls1To2         int    `json:"calls_1_to_2"`
	Calls2To1         int    `json:"calls_2_to_1"`
	SharedFieldAccess int    `json:"shared_field_access"`
}
