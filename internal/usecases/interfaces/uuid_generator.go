package interfaces

type UUIDGenerator interface {
	Generate() (string, error)
}
