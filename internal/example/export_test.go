package example

import "go.uber.org/zap"

// NewDependenciesForTest exposes the package-private injection seam only to
// external tests. It is compiled into test builds, not production binaries.
func NewDependenciesForTest(logger *zap.Logger, storage store) *dependencies {
	return newDependencies(logger, storage)
}
