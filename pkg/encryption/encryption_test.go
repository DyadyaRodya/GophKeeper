package encryption

import (
	"reflect"
	"testing"
)

func TestEncryptService(t *testing.T) {
	pass := "testpassword"
	service := &EncryptService{}

	dek, err := service.GenerateDEK()
	if err != nil {
		panic(err)
	}

	kekSalt, err := service.GenerateSalt(16)
	if err != nil {
		panic(err)
	}

	kek := service.GenerateKEK(pass, kekSalt)

	recoveryKey, err := service.GenerateRecoveryKey()
	if err != nil {
		panic(err)
	}

	dekEncrypted, err := service.Encrypt(dek, kek)
	if err != nil {
		panic(err)
	}

	dekDecrypted, err := service.Decrypt(dekEncrypted, kek)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(dek, dekDecrypted) {
		t.Error("dek != dekDecrypted")
	}

	dekEncryptedRecovered, err := service.Encrypt(dek, recoveryKey)
	if err != nil {
		panic(err)
	}

	dekRecovered, err := service.Decrypt(dekEncryptedRecovered, recoveryKey)
	if err != nil {
		panic(err)
	}

	if !reflect.DeepEqual(dek, dekRecovered) {
		t.Error("dek != dekRecovered")
	}

	someData := []byte("some data")
	encryptedData, err := service.Encrypt(someData, dek)
	if err != nil {
		panic(err)
	}
	decryptedData, err := service.Decrypt(encryptedData, dek)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(someData, decryptedData) {
		t.Error("someData != decryptedData")
	}
}
