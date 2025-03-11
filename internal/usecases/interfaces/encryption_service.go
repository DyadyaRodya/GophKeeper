package interfaces

type (
	KEKGenerator interface {
		GenerateKEK(password string, salt []byte) []byte
	}
	DEKGenerator interface {
		GenerateDEK() ([]byte, error)
	}
	SaltGenerator interface {
		GenerateSalt(saltSize int) ([]byte, error)
	}
	RecoveryKeyGenerator interface {
		GenerateRecoveryKey() ([]byte, error)
	}
	Encryptor interface {
		Encrypt(data, key []byte) ([]byte, error)
	}
	Decryptor interface {
		Decrypt(ciphertext, key []byte) ([]byte, error)
	}
)
