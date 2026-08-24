package foo

import "brook/modules/example"

// Foo wraps an Example created via example.Service, standing in for a
// module that owns its own domain logic but delegates part of the work
// to a sibling module in-process.
type Foo struct {
	Example *example.Example `json:"example"`
}
