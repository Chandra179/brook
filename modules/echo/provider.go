package echo

import "brook/modules/example"

// compile-time check: *Dependencies implements example.Provider
var _ example.Provider = (*Dependencies)(nil)

func (d *Dependencies) HandleProvider(r string) {
	d.Logger.Info("echo provider: " + r)
}
