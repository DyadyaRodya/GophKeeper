package dto

import domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"

// Keys dto for server side register, recover and update password results
//
// UserUUID uuid of user-owner of keys
// DEKCiphertext is DEK encrypted by some KEK
// KEKSalt is salt required for generating this KEK
//
// RecoveryKey is recovery key used for DEK encryption on the server side.
// Required for DEK recovery in case when password used for KEK generation has been forgotten.
type Keys struct {
	UserUUID      string
	DEKCiphertext []byte
	KEKSalt       []byte
	RecoveryKey   []byte
}

func (k *Keys) ToShort() *domainmodels.ShortKeyInfo {
	return &domainmodels.ShortKeyInfo{
		UserUUID:      k.UserUUID,
		DEKCiphertext: k.DEKCiphertext,
		KEKSalt:       k.KEKSalt,
	}
}

// ClientLoginResult DTO for client side login result
type ClientLoginResult struct {
	UserUUID string
	DEK      []byte
}

// ClientRegisterResult DTO for client side register result
type ClientRegisterResult struct {
	UserUUID    string
	DEK         []byte
	RecoveryKey []byte
}

// ClientRecoverResult DTO for client side recover result
type ClientRecoverResult struct {
	UserUUID    string
	DEK         []byte
	RecoveryKey []byte
}
