package example

import "go.uber.org/zap"

// NewDependenciesForTest exposes the private storage seam to external tests.
func NewDependenciesForTest(logger *zap.Logger, storage store) *dependencies {
	return newDependencies(logger, storage)
}
