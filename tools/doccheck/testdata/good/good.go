// Package good is fully documented.
package good

// T is a type.
type T struct{}

// M is a method.
func (T) M() {}

// G is generic.
func G[X any](x X) X { return x }
