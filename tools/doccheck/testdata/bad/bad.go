package bad

type T[X any] struct{}

func (*T[X]) M() {}

func init() {}

func F() {}
