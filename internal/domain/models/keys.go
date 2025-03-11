package models

// ShortKeyInfo domain model for client
//
// UserUUID uuid of user-owner of keys
// DEKCiphertext is DEK encrypted by some KEK
// KEKSalt is salt required for generating this KEK
type ShortKeyInfo struct {
	UserUUID      string `json:"user_uuid"`
	DEKCiphertext []byte `json:"dek_ciphertext"`
	KEKSalt       []byte `json:"kek_salt"`
}

// KeysInfo domain model for server
//
// DEKCiphertext is DEK encrypted by some KEK
// KEKSalt is salt required for generating this KEK
// RecoveryDEKCiphertext is DEK encrypted by some recovery key
type KeysInfo struct {
	DEKCiphertext         []byte
	KEKSalt               []byte
	RecoveryDEKCiphertext []byte
}

// ToShort converts *KeysInfo to new *ShortKeyInfo
func (i *KeysInfo) ToShort(userUUID string) *ShortKeyInfo {
	return &ShortKeyInfo{
		UserUUID:      userUUID,
		DEKCiphertext: i.DEKCiphertext,
		KEKSalt:       i.KEKSalt,
	}
}
