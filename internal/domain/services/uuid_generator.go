package services

import uuidPkg "github.com/google/uuid"

// UUID4Generator service for generating
type UUID4Generator struct{}

// NewUUID4Generator constructor for UUID4Generator
func NewUUID4Generator() *UUID4Generator {
	return &UUID4Generator{}
}

// Generate generates new random UUID version 4
func (u *UUID4Generator) Generate() (string, error) {
	uuid, err := uuidPkg.NewRandom()
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}
