package meta

type Cloneable interface {
	Clone() Cloneable
}
