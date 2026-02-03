package detector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"quality-bot/src/config"
	"quality-bot/src/model"
	"quality-bot/src/service/codeapi"
	"quality-bot/src/service/metrics"
	"quality-bot/src/util"
)

// Runner manages and runs all detectors.
// It handles detector registration, parallel execution, and result aggregation.
type Runner struct {
	detectors []Detector
	cfg       *config.Config
}

// NewRunner creates a new detector runner with all detectors registered
func NewRunner(metricsProvider *metrics.Provider, codeapiClient *codeapi.Client, cfg *config.Config) *Runner {
	base := NewBaseDetector(metricsProvider, cfg)

	detectors := []Detector{
		NewComplexityDetector(base, cfg.Detectors.Complexity),
		NewSizeAndStructureDetector(base, cfg.Detectors.SizeAndStructure),
		NewCouplingDetector(base, cfg.Detectors.Coupling),
		NewDuplicationDetector(base, cfg.Detectors.Duplication, metricsProvider, codeapiClient),
		// DeadCodeDetector - planned for future release
	}

	util.Debug("Detector runner initialized with %d detectors", len(detectors))
	for _, d := range detectors {
		status := "disabled"
		if d.IsEnabled() {
			status = "enabled"
		}
		util.Debug("  - %s: %s", d.Name(), status)
	}

	return &Runner{
		detectors: detectors,
		cfg:       cfg,
	}
}

// RunAll executes all enabled detectors and returns combined issues
func (r *Runner) RunAll(ctx context.Context) ([]model.DebtIssue, error) {
	startTime := time.Now()
	util.Info("Starting debt detection")

	var (
		allIssues []model.DebtIssue
		mu        sync.Mutex
		wg        sync.WaitGroup
		errChan   = make(chan error, len(r.detectors))
		sem       = make(chan struct{}, r.cfg.Concurrency.MaxParallelDetectors)
	)

	enabledCount := 0
	for _, d := range r.detectors {
		if !d.IsEnabled() {
			util.Debug("Skipping disabled detector: %s", d.Name())
			continue
		}
		enabledCount++

		wg.Add(1)
		go func(detector Detector) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			detectorStart := time.Now()
			util.Debug("Running detector: %s", detector.Name())

			issues, err := detector.Detect(ctx)
			if err != nil {
				util.Error("Detector %s failed: %v", detector.Name(), err)
				if r.cfg.Detectors.FailFast {
					errChan <- fmt.Errorf("detector %s: %w", detector.Name(), err)
				}
				return
			}

			util.Info("Detector %s found %d issues (took %v)", detector.Name(), len(issues), time.Since(detectorStart))

			mu.Lock()
			allIssues = append(allIssues, issues...)
			mu.Unlock()
		}(d)
	}

	util.Debug("Running %d enabled detectors (max parallel: %d)", enabledCount, r.cfg.Concurrency.MaxParallelDetectors)

	wg.Wait()
	close(errChan)

	// Check for errors
	if err, ok := <-errChan; ok {
		util.Error("Detection aborted due to error: %v", err)
		return nil, err
	}

	util.Info("Detection complete: %d total issues found (took %v)", len(allIssues), time.Since(startTime))
	return allIssues, nil
}

// GetDetector returns a detector by name
func (r *Runner) GetDetector(name string) Detector {
	for _, d := range r.detectors {
		if d.Name() == name {
			return d
		}
	}
	return nil
}

// ListDetectors returns names of all registered detectors
func (r *Runner) ListDetectors() []string {
	names := make([]string, len(r.detectors))
	for i, d := range r.detectors {
		names[i] = d.Name()
	}
	return names
}
