package interfaces

type (
	PasswordHashGenerator interface {
		Hash(password, salt string) (string, error)
	}
	PasswordValidator interface {
		Validate(password string) bool
	}
	PasswordSaltGenerator interface {
		GenerateRandomSalt(saltSize int) (string, error)
	}
	PasswordComparator interface {
		Compare(currPassword, hashedPassword, salt string) (bool, error)
	}
)
