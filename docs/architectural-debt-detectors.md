# Architectural Debt Detectors

This document describes how to implement detectors for architectural-level technical debt. Unlike code smells which operate at the function/class level, architectural debt involves patterns across packages, modules, and system boundaries.

## Overview

Architectural debt detectors analyze higher-level structural issues:

| Detector | Description | Complexity |
|----------|-------------|------------|
| Layering Violations | Forbidden cross-layer dependencies (e.g., UI → DB) | Medium |
| Missing Abstractions | High fan-in concrete classes that should be interfaces | Medium |
| Tight Package Coupling | Bidirectional dependencies between packages | Medium |
| Monolithic Code | Oversized packages with low cohesion | Low |
| Inconsistent Patterns | Similar code following different conventions | High |

---

## Prerequisites

### Current State

The codebase currently operates at **file/class/function level** with no explicit package abstraction. However, the graph model supports cross-class queries via `ClassPairMetrics`, which can be extended for package-level analysis.

### New Category

Add to `src/model/issue.go`:

```go
const (
    CategoryComplexity   Category = "complexity"
    CategorySize         Category = "size"
    CategoryCoupling     Category = "coupling"
    CategoryDuplication  Category = "duplication"
    CategoryDeadCode     Category = "dead_code"
    CategoryArchitecture Category = "architecture"  // NEW
)
```

### New Metrics

Add to `src/model/metrics.go`:

```go
// PackageMetrics contains metrics for a logical package/module
type PackageMetrics struct {
    Name             string  `json:"name"`
    Path             string  `json:"path"`
    FileCount        int     `json:"file_count"`
    ClassCount       int     `json:"class_count"`
    FunctionCount    int     `json:"function_count"`
    InternalCalls    int     `json:"internal_calls"`    // calls within package
    ExternalCalls    int     `json:"external_calls"`    // calls to other packages
    Afferent         int     `json:"afferent"`          // incoming dependencies (Ca)
    Efferent         int     `json:"efferent"`          // outgoing dependencies (Ce)
    Instability      float64 `json:"instability"`       // Ce / (Ca + Ce)
    Cohesion         float64 `json:"cohesion"`          // internal/total coupling ratio
}

// PackagePairMetrics contains coupling metrics between two packages
type PackagePairMetrics struct {
    Package1    string `json:"package1"`
    Package2    string `json:"package2"`
    Calls1To2   int    `json:"calls_1_to_2"`
    Calls2To1   int    `json:"calls_2_to_1"`
    SharedTypes int    `json:"shared_types"`
}
```

---

## 1. Layering Violations Detector

Detects when code in one architectural layer directly accesses code in a forbidden layer (e.g., UI components directly querying the database).

### Configuration

Add to `src/config/config.go`:

```go
// LayeringDetectorConfig contains settings for layer violation detection
type LayeringDetectorConfig struct {
    Enabled bool              `yaml:"enabled"`
    Layers  []LayerDefinition `yaml:"layers"`
    Rules   []LayerRule       `yaml:"rules"`
}

// LayerDefinition defines an architectural layer
type LayerDefinition struct {
    Name     string   `yaml:"name"`     // e.g., "ui", "service", "data"
    Patterns []string `yaml:"patterns"` // glob patterns matching this layer
}

// LayerRule defines forbidden dependencies between layers
type LayerRule struct {
    From      string   `yaml:"from"`      // source layer name
    Forbidden []string `yaml:"forbidden"` // layers that 'from' cannot depend on
}
```

Add to `DetectorsConfig`:

```go
type DetectorsConfig struct {
    // ... existing detectors ...
    Layering LayeringDetectorConfig `yaml:"layering"`
}
```

### Default Configuration

Add to `src/config/defaults.go`:

```go
Layering: LayeringDetectorConfig{
    Enabled: true,
    Layers: []LayerDefinition{
        {Name: "handler", Patterns: []string{"src/handler/**", "**/handler/**"}},
        {Name: "controller", Patterns: []string{"src/controller/**", "**/controller/**"}},
        {Name: "service", Patterns: []string{"src/service/**", "**/service/**"}},
        {Name: "repository", Patterns: []string{"src/repository/**", "**/repo/**", "**/db/**"}},
    },
    Rules: []LayerRule{
        {From: "handler", Forbidden: []string{"repository"}},
        {From: "repository", Forbidden: []string{"handler", "controller"}},
    },
},
```

### Example YAML Configuration

```yaml
detectors:
  layering:
    enabled: true
    layers:
      - name: ui
        patterns:
          - "src/ui/**"
          - "src/web/**"
          - "src/handler/**"
      - name: service
        patterns:
          - "src/service/**"
          - "src/controller/**"
      - name: data
        patterns:
          - "src/repository/**"
          - "src/db/**"
          - "src/dao/**"
    rules:
      - from: ui
        forbidden: [data]      # UI cannot directly access data layer
      - from: data
        forbidden: [ui]        # Data layer cannot depend on UI
```

### Implementation

Create `src/service/detector/layering.go`:

```go
package detector

import (
    "context"
    "fmt"
    "path/filepath"

    "quality-bot/src/config"
    "quality-bot/src/model"
)

// LayeringDetector detects architectural layer violations
type LayeringDetector struct {
    BaseDetector
    cfg          config.LayeringDetectorConfig
    layerLookup  map[string]string // maps file patterns to layer names
    forbiddenMap map[string]map[string]bool // from -> set of forbidden layers
}

// NewLayeringDetector creates a new layering detector
func NewLayeringDetector(base BaseDetector, cfg config.LayeringDetectorConfig) *LayeringDetector {
    d := &LayeringDetector{
        BaseDetector: base,
        cfg:          cfg,
        layerLookup:  make(map[string]string),
        forbiddenMap: make(map[string]map[string]bool),
    }
    d.buildLookups()
    return d
}

func (d *LayeringDetector) buildLookups() {
    // Build forbidden map for O(1) lookups
    for _, rule := range d.cfg.Rules {
        d.forbiddenMap[rule.From] = make(map[string]bool)
        for _, forbidden := range rule.Forbidden {
            d.forbiddenMap[rule.From][forbidden] = true
        }
    }
}

func (d *LayeringDetector) Name() string {
    return "layering"
}

func (d *LayeringDetector) IsEnabled() bool {
    return d.cfg.Enabled
}

func (d *LayeringDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    // Get all class pair metrics (cross-class calls)
    pairs, err := d.Metrics.GetClassPairMetrics(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetching class pair metrics: %w", err)
    }

    var issues []model.DebtIssue
    seen := make(map[string]bool) // deduplicate violations

    for _, pair := range pairs {
        // Only check pairs with actual calls
        if pair.Calls1To2 == 0 && pair.Calls2To1 == 0 {
            continue
        }

        srcLayer := d.getLayer(pair.Class1File)
        dstLayer := d.getLayer(pair.Class2File)

        // Skip if either class is not in a defined layer
        if srcLayer == "" || dstLayer == "" {
            continue
        }

        // Check direction 1 -> 2
        if pair.Calls1To2 > 0 && d.isForbidden(srcLayer, dstLayer) {
            key := fmt.Sprintf("%s->%s:%s->%s", srcLayer, dstLayer, pair.Class1Name, pair.Class2Name)
            if !seen[key] {
                seen[key] = true
                issues = append(issues, d.createIssue(pair, srcLayer, dstLayer, pair.Calls1To2, true))
            }
        }

        // Check direction 2 -> 1
        if pair.Calls2To1 > 0 && d.isForbidden(dstLayer, srcLayer) {
            key := fmt.Sprintf("%s->%s:%s->%s", dstLayer, srcLayer, pair.Class2Name, pair.Class1Name)
            if !seen[key] {
                seen[key] = true
                issues = append(issues, d.createIssue(pair, dstLayer, srcLayer, pair.Calls2To1, false))
            }
        }
    }

    return d.FilterBySeverity(issues), nil
}

func (d *LayeringDetector) getLayer(filePath string) string {
    for _, layer := range d.cfg.Layers {
        for _, pattern := range layer.Patterns {
            if matched, _ := filepath.Match(pattern, filePath); matched {
                return layer.Name
            }
            // Also try with ** glob support
            if matchGlob(pattern, filePath) {
                return layer.Name
            }
        }
    }
    return ""
}

func (d *LayeringDetector) isForbidden(from, to string) bool {
    if forbidden, ok := d.forbiddenMap[from]; ok {
        return forbidden[to]
    }
    return false
}

func (d *LayeringDetector) createIssue(pair model.ClassPairMetrics, srcLayer, dstLayer string, callCount int, direction1To2 bool) model.DebtIssue {
    var srcClass, dstClass, srcFile, dstFile string
    if direction1To2 {
        srcClass, dstClass = pair.Class1Name, pair.Class2Name
        srcFile, dstFile = pair.Class1File, pair.Class2File
    } else {
        srcClass, dstClass = pair.Class2Name, pair.Class1Name
        srcFile, dstFile = pair.Class2File, pair.Class1File
    }

    severity := model.SeverityHigh
    if callCount > 5 {
        severity = model.SeverityCritical
    }

    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "layering_violation",
        Severity:    severity,
        FilePath:    srcFile,
        EntityName:  fmt.Sprintf("%s → %s", srcClass, dstClass),
        EntityType:  "class_pair",
        Description: fmt.Sprintf(
            "Layer violation: %s layer (%s) directly accesses %s layer (%s) with %d calls",
            srcLayer, srcClass, dstLayer, dstClass, callCount,
        ),
        Metrics: map[string]any{
            "source_layer":  srcLayer,
            "target_layer":  dstLayer,
            "source_class":  srcClass,
            "target_class":  dstClass,
            "target_file":   dstFile,
            "call_count":    callCount,
        },
        Suggestion: fmt.Sprintf(
            "Introduce an abstraction in the %s layer that %s can depend on, or route through the %s layer",
            dstLayer, srcLayer, "service",
        ),
    }
}

// matchGlob handles ** glob patterns
func matchGlob(pattern, path string) bool {
    // Simple implementation - for production, use doublestar library
    // This handles common cases like "src/handler/**"
    if len(pattern) > 2 && pattern[len(pattern)-2:] == "**" {
        prefix := pattern[:len(pattern)-2]
        return len(path) >= len(prefix) && path[:len(prefix)] == prefix
    }
    matched, _ := filepath.Match(pattern, path)
    return matched
}
```

### Output Example

```json
{
  "category": "architecture",
  "subcategory": "layering_violation",
  "severity": "high",
  "file_path": "src/handler/user_handler.go",
  "entity_name": "UserHandler → UserRepository",
  "entity_type": "class_pair",
  "description": "Layer violation: handler layer (UserHandler) directly accesses repository layer (UserRepository) with 3 calls",
  "metrics": {
    "source_layer": "handler",
    "target_layer": "repository",
    "source_class": "UserHandler",
    "target_class": "UserRepository",
    "call_count": 3
  },
  "suggestion": "Introduce an abstraction in the repository layer that handler can depend on, or route through the service layer"
}
```

---

## 2. Missing Abstractions Detector

Detects concrete classes with high fan-in (many dependents) that should likely be interfaces.

### Configuration

Add to `src/config/config.go`:

```go
// AbstractionDetectorConfig contains settings for missing abstraction detection
type AbstractionDetectorConfig struct {
    Enabled            bool     `yaml:"enabled"`
    HighFanInThreshold int      `yaml:"high_fan_in_threshold"` // dependents to trigger
    InterfacePatterns  []string `yaml:"interface_patterns"`    // patterns identifying interfaces
}
```

### Defaults

```go
Abstraction: AbstractionDetectorConfig{
    Enabled:            true,
    HighFanInThreshold: 5,
    InterfacePatterns:  []string{"*Interface", "*I", "I*"},
},
```

### Implementation

Create `src/service/detector/abstraction.go`:

```go
package detector

import (
    "context"
    "fmt"
    "strings"

    "quality-bot/src/config"
    "quality-bot/src/model"
)

// AbstractionDetector detects missing abstractions
type AbstractionDetector struct {
    BaseDetector
    cfg config.AbstractionDetectorConfig
}

func NewAbstractionDetector(base BaseDetector, cfg config.AbstractionDetectorConfig) *AbstractionDetector {
    return &AbstractionDetector{
        BaseDetector: base,
        cfg:          cfg,
    }
}

func (d *AbstractionDetector) Name() string {
    return "abstraction"
}

func (d *AbstractionDetector) IsEnabled() bool {
    return d.cfg.Enabled
}

func (d *AbstractionDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    classes, err := d.Metrics.GetAllClassMetrics(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetching class metrics: %w", err)
    }

    var issues []model.DebtIssue

    for _, cls := range classes {
        if d.ShouldExclude(cls.FilePath, cls.Name, "") {
            continue
        }

        // Skip if already an interface
        if d.isInterface(cls.Name) {
            continue
        }

        // High fan-in on concrete class suggests missing abstraction
        if cls.DependentCount >= d.cfg.HighFanInThreshold {
            issues = append(issues, d.createMissingAbstractionIssue(cls))
        }
    }

    return d.FilterBySeverity(issues), nil
}

func (d *AbstractionDetector) isInterface(name string) bool {
    // Check common interface naming patterns
    for _, pattern := range d.cfg.InterfacePatterns {
        if strings.HasPrefix(pattern, "*") && strings.HasSuffix(name, pattern[1:]) {
            return true
        }
        if strings.HasSuffix(pattern, "*") && strings.HasPrefix(name, pattern[:len(pattern)-1]) {
            return true
        }
        if name == pattern {
            return true
        }
    }

    // Language-specific heuristics
    lowerName := strings.ToLower(name)
    return strings.HasSuffix(lowerName, "interface") ||
           strings.HasPrefix(lowerName, "i") && len(name) > 1 && name[1] >= 'A' && name[1] <= 'Z'
}

func (d *AbstractionDetector) createMissingAbstractionIssue(cls model.ClassMetrics) model.DebtIssue {
    severity := model.SeverityMedium
    if cls.DependentCount >= d.cfg.HighFanInThreshold*2 {
        severity = model.SeverityHigh
    }

    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "missing_abstraction",
        Severity:    severity,
        FilePath:    cls.FilePath,
        StartLine:   cls.StartLine,
        EndLine:     cls.EndLine,
        EntityName:  cls.Name,
        EntityType:  "class",
        Description: fmt.Sprintf(
            "Concrete class %s has %d dependents - consider extracting an interface",
            cls.Name, cls.DependentCount,
        ),
        Metrics: map[string]any{
            "dependent_count":  cls.DependentCount,
            "method_count":     cls.MethodCount,
            "dependency_count": cls.DependencyCount,
        },
        Suggestion: "Extract a public interface for this class's methods and have dependents use the interface instead",
    }
}
```

---

## 3. Tight Package Coupling Detector

Detects packages/modules with excessive bidirectional dependencies.

### Configuration

```go
// PackageCouplingDetectorConfig contains settings for package coupling detection
type PackageCouplingDetectorConfig struct {
    Enabled                bool `yaml:"enabled"`
    BidirectionalThreshold int  `yaml:"bidirectional_threshold"` // min calls each direction
    TotalCallThreshold     int  `yaml:"total_call_threshold"`    // min total cross-pkg calls
    PackageDepth           int  `yaml:"package_depth"`           // path segments for package grouping
}
```

### Defaults

```go
PackageCoupling: PackageCouplingDetectorConfig{
    Enabled:                true,
    BidirectionalThreshold: 3,
    TotalCallThreshold:     10,
    PackageDepth:           3, // e.g., "src/service/user" = 3 segments
},
```

### Metrics Provider Extension

Add to `src/service/metrics/provider.go`:

```go
// GetPackagePairMetrics aggregates class pair metrics at package level
func (p *Provider) GetPackagePairMetrics(ctx context.Context, depth int) ([]model.PackagePairMetrics, error) {
    classPairs, err := p.GetClassPairMetrics(ctx)
    if err != nil {
        return nil, err
    }

    // Aggregate by package pairs
    pkgPairs := make(map[string]*model.PackagePairMetrics)

    for _, pair := range classPairs {
        pkg1 := extractPackage(pair.Class1File, depth)
        pkg2 := extractPackage(pair.Class2File, depth)

        if pkg1 == pkg2 {
            continue // Skip intra-package calls
        }

        // Normalize key (smaller package name first)
        key := pkg1 + "|" + pkg2
        if pkg1 > pkg2 {
            key = pkg2 + "|" + pkg1
        }

        if _, ok := pkgPairs[key]; !ok {
            pkgPairs[key] = &model.PackagePairMetrics{
                Package1: pkg1,
                Package2: pkg2,
            }
        }

        pp := pkgPairs[key]
        if pkg1 == pp.Package1 {
            pp.Calls1To2 += pair.Calls1To2
            pp.Calls2To1 += pair.Calls2To1
        } else {
            pp.Calls1To2 += pair.Calls2To1
            pp.Calls2To1 += pair.Calls1To2
        }
    }

    result := make([]model.PackagePairMetrics, 0, len(pkgPairs))
    for _, pp := range pkgPairs {
        result = append(result, *pp)
    }

    return result, nil
}

// extractPackage extracts package path from file path
func extractPackage(filePath string, depth int) string {
    parts := strings.Split(filePath, "/")
    if len(parts) <= depth {
        // Return directory portion
        if len(parts) > 1 {
            return strings.Join(parts[:len(parts)-1], "/")
        }
        return parts[0]
    }
    return strings.Join(parts[:depth], "/")
}
```

### Implementation

Create `src/service/detector/package_coupling.go`:

```go
package detector

import (
    "context"
    "fmt"

    "quality-bot/src/config"
    "quality-bot/src/model"
)

// PackageCouplingDetector detects tightly coupled packages
type PackageCouplingDetector struct {
    BaseDetector
    cfg config.PackageCouplingDetectorConfig
}

func NewPackageCouplingDetector(base BaseDetector, cfg config.PackageCouplingDetectorConfig) *PackageCouplingDetector {
    return &PackageCouplingDetector{
        BaseDetector: base,
        cfg:          cfg,
    }
}

func (d *PackageCouplingDetector) Name() string {
    return "package_coupling"
}

func (d *PackageCouplingDetector) IsEnabled() bool {
    return d.cfg.Enabled
}

func (d *PackageCouplingDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    pairs, err := d.Metrics.GetPackagePairMetrics(ctx, d.cfg.PackageDepth)
    if err != nil {
        return nil, fmt.Errorf("fetching package pair metrics: %w", err)
    }

    var issues []model.DebtIssue

    for _, pair := range pairs {
        totalCalls := pair.Calls1To2 + pair.Calls2To1

        // Check for bidirectional tight coupling
        if pair.Calls1To2 >= d.cfg.BidirectionalThreshold &&
           pair.Calls2To1 >= d.cfg.BidirectionalThreshold {
            issues = append(issues, d.createBidirectionalIssue(pair))
            continue
        }

        // Check for high total coupling (even if mostly unidirectional)
        if totalCalls >= d.cfg.TotalCallThreshold {
            issues = append(issues, d.createHighCouplingIssue(pair))
        }
    }

    return d.FilterBySeverity(issues), nil
}

func (d *PackageCouplingDetector) createBidirectionalIssue(pair model.PackagePairMetrics) model.DebtIssue {
    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "tight_coupling",
        Severity:    model.SeverityHigh,
        EntityName:  fmt.Sprintf("%s ↔ %s", pair.Package1, pair.Package2),
        EntityType:  "package_pair",
        Description: fmt.Sprintf(
            "Packages are tightly coupled with bidirectional dependencies: %d calls %s→%s, %d calls %s→%s",
            pair.Calls1To2, pair.Package1, pair.Package2,
            pair.Calls2To1, pair.Package2, pair.Package1,
        ),
        Metrics: map[string]any{
            "package1":     pair.Package1,
            "package2":     pair.Package2,
            "calls_1_to_2": pair.Calls1To2,
            "calls_2_to_1": pair.Calls2To1,
            "total_calls":  pair.Calls1To2 + pair.Calls2To1,
        },
        Suggestion: "Consider introducing a shared interface or extracting common functionality to break the circular dependency",
    }
}

func (d *PackageCouplingDetector) createHighCouplingIssue(pair model.PackagePairMetrics) model.DebtIssue {
    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "high_package_coupling",
        Severity:    model.SeverityMedium,
        EntityName:  fmt.Sprintf("%s → %s", pair.Package1, pair.Package2),
        EntityType:  "package_pair",
        Description: fmt.Sprintf(
            "High coupling between packages: %d total cross-package calls",
            pair.Calls1To2+pair.Calls2To1,
        ),
        Metrics: map[string]any{
            "package1":     pair.Package1,
            "package2":     pair.Package2,
            "calls_1_to_2": pair.Calls1To2,
            "calls_2_to_1": pair.Calls2To1,
            "total_calls":  pair.Calls1To2 + pair.Calls2To1,
        },
        Suggestion: "Review the dependency direction; consider if one package should own the shared functionality",
    }
}
```

---

## 4. Monolithic Code Detector

Detects oversized packages with low cohesion that should be split.

### Configuration

```go
// MonolithDetectorConfig contains settings for monolith detection
type MonolithDetectorConfig struct {
    Enabled           bool    `yaml:"enabled"`
    MaxPackageClasses int     `yaml:"max_package_classes"` // max classes per package
    MaxPackageFiles   int     `yaml:"max_package_files"`   // max files per package
    MinCohesionRatio  float64 `yaml:"min_cohesion_ratio"`  // internal/total calls
    PackageDepth      int     `yaml:"package_depth"`       // path depth for grouping
}
```

### Defaults

```go
Monolith: MonolithDetectorConfig{
    Enabled:           true,
    MaxPackageClasses: 20,
    MaxPackageFiles:   15,
    MinCohesionRatio:  0.3,
    PackageDepth:      3,
},
```

### Metrics Provider Extension

```go
// GetPackageMetrics aggregates metrics at package level
func (p *Provider) GetPackageMetrics(ctx context.Context, depth int) ([]model.PackageMetrics, error) {
    classes, err := p.GetAllClassMetrics(ctx)
    if err != nil {
        return nil, err
    }

    files, err := p.GetAllFileMetrics(ctx)
    if err != nil {
        return nil, err
    }

    classPairs, err := p.GetClassPairMetrics(ctx)
    if err != nil {
        return nil, err
    }

    // Aggregate by package
    packages := make(map[string]*model.PackageMetrics)

    // Count classes per package
    for _, cls := range classes {
        pkg := extractPackage(cls.FilePath, depth)
        if _, ok := packages[pkg]; !ok {
            packages[pkg] = &model.PackageMetrics{Name: pkg, Path: pkg}
        }
        packages[pkg].ClassCount++
        packages[pkg].FunctionCount += cls.MethodCount
    }

    // Count files per package
    for _, f := range files {
        pkg := extractPackage(f.Path, depth)
        if _, ok := packages[pkg]; !ok {
            packages[pkg] = &model.PackageMetrics{Name: pkg, Path: pkg}
        }
        packages[pkg].FileCount++
    }

    // Calculate internal vs external calls
    for _, pair := range classPairs {
        pkg1 := extractPackage(pair.Class1File, depth)
        pkg2 := extractPackage(pair.Class2File, depth)

        totalCalls := pair.Calls1To2 + pair.Calls2To1

        if pkg1 == pkg2 {
            // Internal calls
            if p, ok := packages[pkg1]; ok {
                p.InternalCalls += totalCalls
            }
        } else {
            // External calls
            if p, ok := packages[pkg1]; ok {
                p.ExternalCalls += pair.Calls1To2
                p.Efferent += pair.Calls1To2
                p.Afferent += pair.Calls2To1
            }
            if p, ok := packages[pkg2]; ok {
                p.ExternalCalls += pair.Calls2To1
                p.Efferent += pair.Calls2To1
                p.Afferent += pair.Calls1To2
            }
        }
    }

    // Calculate derived metrics
    result := make([]model.PackageMetrics, 0, len(packages))
    for _, pkg := range packages {
        totalCalls := pkg.InternalCalls + pkg.ExternalCalls
        if totalCalls > 0 {
            pkg.Cohesion = float64(pkg.InternalCalls) / float64(totalCalls)
        }
        totalDeps := pkg.Afferent + pkg.Efferent
        if totalDeps > 0 {
            pkg.Instability = float64(pkg.Efferent) / float64(totalDeps)
        }
        result = append(result, *pkg)
    }

    return result, nil
}
```

### Implementation

Create `src/service/detector/monolith.go`:

```go
package detector

import (
    "context"
    "fmt"

    "quality-bot/src/config"
    "quality-bot/src/model"
)

// MonolithDetector detects oversized packages with low cohesion
type MonolithDetector struct {
    BaseDetector
    cfg config.MonolithDetectorConfig
}

func NewMonolithDetector(base BaseDetector, cfg config.MonolithDetectorConfig) *MonolithDetector {
    return &MonolithDetector{
        BaseDetector: base,
        cfg:          cfg,
    }
}

func (d *MonolithDetector) Name() string {
    return "monolith"
}

func (d *MonolithDetector) IsEnabled() bool {
    return d.cfg.Enabled
}

func (d *MonolithDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    packages, err := d.Metrics.GetPackageMetrics(ctx, d.cfg.PackageDepth)
    if err != nil {
        return nil, fmt.Errorf("fetching package metrics: %w", err)
    }

    var issues []model.DebtIssue

    for _, pkg := range packages {
        // Check for oversized package (too many classes)
        if pkg.ClassCount > d.cfg.MaxPackageClasses {
            issues = append(issues, d.createOversizedClassIssue(pkg))
        }

        // Check for oversized package (too many files)
        if pkg.FileCount > d.cfg.MaxPackageFiles {
            issues = append(issues, d.createOversizedFileIssue(pkg))
        }

        // Check for low cohesion (only if package has enough classes to matter)
        if pkg.ClassCount >= 3 && pkg.Cohesion < d.cfg.MinCohesionRatio {
            totalCalls := pkg.InternalCalls + pkg.ExternalCalls
            if totalCalls > 0 { // Only report if there are actual calls
                issues = append(issues, d.createLowCohesionIssue(pkg))
            }
        }
    }

    return d.FilterBySeverity(issues), nil
}

func (d *MonolithDetector) createOversizedClassIssue(pkg model.PackageMetrics) model.DebtIssue {
    severity := model.SeverityMedium
    if pkg.ClassCount > d.cfg.MaxPackageClasses*2 {
        severity = model.SeverityHigh
    }

    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "monolithic_package",
        Severity:    severity,
        EntityName:  pkg.Name,
        EntityType:  "package",
        Description: fmt.Sprintf(
            "Package %s has %d classes (threshold: %d) - consider splitting into smaller modules",
            pkg.Name, pkg.ClassCount, d.cfg.MaxPackageClasses,
        ),
        Metrics: map[string]any{
            "class_count":    pkg.ClassCount,
            "file_count":     pkg.FileCount,
            "function_count": pkg.FunctionCount,
            "threshold":      d.cfg.MaxPackageClasses,
        },
        Suggestion: "Identify cohesive subsets of classes and extract them into separate packages",
    }
}

func (d *MonolithDetector) createOversizedFileIssue(pkg model.PackageMetrics) model.DebtIssue {
    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "monolithic_package",
        Severity:    model.SeverityMedium,
        EntityName:  pkg.Name,
        EntityType:  "package",
        Description: fmt.Sprintf(
            "Package %s has %d files (threshold: %d) - consider splitting",
            pkg.Name, pkg.FileCount, d.cfg.MaxPackageFiles,
        ),
        Metrics: map[string]any{
            "file_count": pkg.FileCount,
            "threshold":  d.cfg.MaxPackageFiles,
        },
        Suggestion: "Group related files into sub-packages based on their responsibilities",
    }
}

func (d *MonolithDetector) createLowCohesionIssue(pkg model.PackageMetrics) model.DebtIssue {
    return model.DebtIssue{
        Category:    model.CategoryArchitecture,
        Subcategory: "low_cohesion",
        Severity:    model.SeverityMedium,
        EntityName:  pkg.Name,
        EntityType:  "package",
        Description: fmt.Sprintf(
            "Package %s has low cohesion (%.1f%%) - classes may not belong together",
            pkg.Name, pkg.Cohesion*100,
        ),
        Metrics: map[string]any{
            "cohesion":       pkg.Cohesion,
            "internal_calls": pkg.InternalCalls,
            "external_calls": pkg.ExternalCalls,
            "class_count":    pkg.ClassCount,
            "threshold":      d.cfg.MinCohesionRatio,
        },
        Suggestion: "Review the package contents - classes with more external than internal dependencies may belong in other packages",
    }
}
```

---

## 5. Inconsistent Patterns Detector

Detects code that follows inconsistent conventions compared to similar code elsewhere.

### Configuration

```go
// PatternDetectorConfig contains settings for pattern consistency detection
type PatternDetectorConfig struct {
    Enabled     bool                `yaml:"enabled"`
    Conventions []ConventionPattern `yaml:"conventions"`
}

// ConventionPattern defines an expected code convention
type ConventionPattern struct {
    Name            string   `yaml:"name"`             // e.g., "handlers"
    ClassPattern    string   `yaml:"class_pattern"`    // regex for class names
    ExpectedMethods []string `yaml:"expected_methods"` // methods these classes should have
    ExpectedPackage string   `yaml:"expected_package"` // glob for expected location
}
```

### Example Configuration

```yaml
detectors:
  patterns:
    enabled: true
    conventions:
      - name: handlers
        class_pattern: ".*Handler$"
        expected_methods: ["Handle", "Validate"]
        expected_package: "src/handler/**"

      - name: repositories
        class_pattern: ".*Repository$"
        expected_methods: ["Find*", "Save", "Delete"]
        expected_package: "src/repository/**"

      - name: services
        class_pattern: ".*Service$"
        expected_package: "src/service/**"
```

### Implementation

Create `src/service/detector/patterns.go`:

```go
package detector

import (
    "context"
    "fmt"
    "regexp"
    "strings"

    "quality-bot/src/config"
    "quality-bot/src/model"
)

// PatternDetector detects inconsistent code patterns
type PatternDetector struct {
    BaseDetector
    cfg              config.PatternDetectorConfig
    compiledPatterns map[string]*regexp.Regexp
}

func NewPatternDetector(base BaseDetector, cfg config.PatternDetectorConfig) *PatternDetector {
    d := &PatternDetector{
        BaseDetector:     base,
        cfg:              cfg,
        compiledPatterns: make(map[string]*regexp.Regexp),
    }
    d.compilePatterns()
    return d
}

func (d *PatternDetector) compilePatterns() {
    for _, conv := range d.cfg.Conventions {
        if conv.ClassPattern != "" {
            if re, err := regexp.Compile(conv.ClassPattern); err == nil {
                d.compiledPatterns[conv.Name] = re
            }
        }
    }
}

func (d *PatternDetector) Name() string {
    return "patterns"
}

func (d *PatternDetector) IsEnabled() bool {
    return d.cfg.Enabled
}

func (d *PatternDetector) Detect(ctx context.Context) ([]model.DebtIssue, error) {
    classes, err := d.Metrics.GetAllClassMetrics(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetching class metrics: %w", err)
    }

    var issues []model.DebtIssue

    // Check each convention
    for _, conv := range d.cfg.Conventions {
        re := d.compiledPatterns[conv.Name]
        if re == nil {
            continue
        }

        for _, cls := range classes {
            if !re.MatchString(cls.Name) {
                continue
            }

            // Class matches the pattern - check if it follows the convention

            // Check package location
            if conv.ExpectedPackage != "" && !matchGlob(conv.ExpectedPackage, cls.FilePath) {
                issues = append(issues, model.DebtIssue{
                    Category:    model.CategoryArchitecture,
                    Subcategory: "inconsistent_location",
                    Severity:    model.SeverityLow,
                    FilePath:    cls.FilePath,
                    StartLine:   cls.StartLine,
                    EndLine:     cls.EndLine,
                    EntityName:  cls.Name,
                    EntityType:  "class",
                    Description: fmt.Sprintf(
                        "Class %s matches '%s' pattern but is not in expected location %s",
                        cls.Name, conv.Name, conv.ExpectedPackage,
                    ),
                    Metrics: map[string]any{
                        "convention":       conv.Name,
                        "expected_package": conv.ExpectedPackage,
                        "actual_path":      cls.FilePath,
                    },
                    Suggestion: fmt.Sprintf("Move %s to %s to follow project conventions", cls.Name, conv.ExpectedPackage),
                })
            }
        }
    }

    // Detect naming inconsistencies (e.g., some use Handler, others use Controller)
    issues = append(issues, d.detectNamingInconsistencies(classes)...)

    return d.FilterBySeverity(issues), nil
}

func (d *PatternDetector) detectNamingInconsistencies(classes []model.ClassMetrics) []model.DebtIssue {
    var issues []model.DebtIssue

    // Group by common suffixes
    suffixGroups := make(map[string][]model.ClassMetrics)
    commonSuffixes := []string{"Handler", "Controller", "Service", "Repository", "Manager", "Helper", "Utils", "Util"}

    for _, cls := range classes {
        for _, suffix := range commonSuffixes {
            if strings.HasSuffix(cls.Name, suffix) {
                // Extract package from path
                pkg := extractPackage(cls.FilePath, 2)
                key := pkg + ":" + suffix
                suffixGroups[key] = append(suffixGroups[key], cls)
                break
            }
        }
    }

    // Check for mixed patterns in same package (e.g., Handler and Controller)
    pkgPatterns := make(map[string][]string) // pkg -> list of suffixes used
    for key := range suffixGroups {
        parts := strings.SplitN(key, ":", 2)
        if len(parts) == 2 {
            pkg, suffix := parts[0], parts[1]
            pkgPatterns[pkg] = append(pkgPatterns[pkg], suffix)
        }
    }

    // Report packages with conflicting patterns
    conflictingSuffixes := map[string]string{
        "Handler":    "Controller",
        "Controller": "Handler",
        "Utils":      "Util",
        "Util":       "Utils",
        "Helper":     "Utils",
    }

    for pkg, suffixes := range pkgPatterns {
        for i, s1 := range suffixes {
            if conflict, ok := conflictingSuffixes[s1]; ok {
                for _, s2 := range suffixes[i+1:] {
                    if s2 == conflict {
                        issues = append(issues, model.DebtIssue{
                            Category:    model.CategoryArchitecture,
                            Subcategory: "inconsistent_naming",
                            Severity:    model.SeverityLow,
                            EntityName:  pkg,
                            EntityType:  "package",
                            Description: fmt.Sprintf(
                                "Package %s uses both %s and %s patterns - choose one for consistency",
                                pkg, s1, s2,
                            ),
                            Metrics: map[string]any{
                                "package":  pkg,
                                "pattern1": s1,
                                "pattern2": s2,
                            },
                            Suggestion: fmt.Sprintf("Standardize on either %s or %s naming convention", s1, s2),
                        })
                    }
                }
            }
        }
    }

    return issues
}
```

---

## Registration

Update `src/service/detector/runner.go` to register the new detectors:

```go
func NewRunner(metricsProvider *metrics.Provider, cfg *config.Config) *Runner {
    base := NewBaseDetector(metricsProvider, cfg)

    detectors := []Detector{
        // Existing detectors
        NewComplexityDetector(base, cfg.Detectors.Complexity),
        NewSizeAndStructureDetector(base, cfg.Detectors.SizeAndStructure),
        NewCouplingDetector(base, cfg.Detectors.Coupling),
        NewDuplicationDetector(base, cfg.Detectors.Duplication, metricsProvider),

        // New architectural detectors
        NewLayeringDetector(base, cfg.Detectors.Layering),
        NewAbstractionDetector(base, cfg.Detectors.Abstraction),
        NewPackageCouplingDetector(base, cfg.Detectors.PackageCoupling),
        NewMonolithDetector(base, cfg.Detectors.Monolith),
        NewPatternDetector(base, cfg.Detectors.Patterns),
    }

    return &Runner{
        detectors: detectors,
        cfg:       cfg,
    }
}
```

---

## Detection Capabilities Matrix

| Detector | Subcategory | Severity | Entity Type |
|----------|-------------|----------|-------------|
| Layering | `layering_violation` | High/Critical | class_pair |
| Abstraction | `missing_abstraction` | Medium/High | class |
| Package Coupling | `tight_coupling` | High | package_pair |
| Package Coupling | `high_package_coupling` | Medium | package_pair |
| Monolith | `monolithic_package` | Medium/High | package |
| Monolith | `low_cohesion` | Medium | package |
| Patterns | `inconsistent_location` | Low | class |
| Patterns | `inconsistent_naming` | Low | package |

---

## Implementation Priority

| Order | Detector | Effort | Value | Prerequisites |
|-------|----------|--------|-------|---------------|
| 1 | Layering Violations | Medium | High | Config-driven layer definitions |
| 2 | Package Coupling | Medium | High | Package-level metrics aggregation |
| 3 | Monolithic Code | Low | Medium | Package metrics (from #2) |
| 4 | Missing Abstractions | Low | Medium | Uses existing ClassMetrics |
| 5 | Inconsistent Patterns | High | Low | Convention configuration |

---

## Future Enhancements

### Circular Dependency Detection

Extend package coupling to detect cycles:

```go
// DetectCycles finds circular dependency chains
func (d *PackageCouplingDetector) DetectCycles(packages []model.PackageMetrics, pairs []model.PackagePairMetrics) [][]string {
    // Build adjacency list
    graph := make(map[string][]string)
    for _, pair := range pairs {
        if pair.Calls1To2 > 0 {
            graph[pair.Package1] = append(graph[pair.Package1], pair.Package2)
        }
        if pair.Calls2To1 > 0 {
            graph[pair.Package2] = append(graph[pair.Package2], pair.Package1)
        }
    }

    // DFS to find cycles
    // ... implementation
}
```

### Stability Metrics (Robert C. Martin)

Calculate package stability and flag violations of the Stable Dependencies Principle:

```go
// Instability = Ce / (Ca + Ce)
// Stable packages (I ≈ 0) should not depend on unstable packages (I ≈ 1)

func (d *PackageCouplingDetector) checkStabilityViolations(packages []model.PackageMetrics, pairs []model.PackagePairMetrics) []model.DebtIssue {
    instability := make(map[string]float64)
    for _, pkg := range packages {
        instability[pkg.Name] = pkg.Instability
    }

    var issues []model.DebtIssue
    for _, pair := range pairs {
        srcInstability := instability[pair.Package1]
        dstInstability := instability[pair.Package2]

        // Stable package depending on unstable package
        if srcInstability < 0.3 && dstInstability > 0.7 && pair.Calls1To2 > 0 {
            issues = append(issues, model.DebtIssue{
                Category:    model.CategoryArchitecture,
                Subcategory: "stability_violation",
                // ...
            })
        }
    }
    return issues
}
```

### Abstractness Metrics

Track the ratio of abstract types (interfaces) to concrete types per package:

```go
// Abstractness = abstract_classes / total_classes
// Combined with Instability, calculate distance from "main sequence"
// D = |A + I - 1| (should be close to 0)
```
