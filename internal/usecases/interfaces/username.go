package interfaces

type UsernameService interface {
	Validate(username string) error
}
