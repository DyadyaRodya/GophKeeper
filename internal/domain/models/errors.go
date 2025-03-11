package models

import "errors"

// internal unexpected errors

var ErrInternalServer = errors.New("internal server error")
var ErrPasswordHashGeneration = errors.Join(ErrInternalServer, errors.New("password hash generation error"))
var ErrSaltGeneration = errors.Join(ErrInternalServer, errors.New("salt generation error"))

// interactor errors

var ErrInteractor = errors.New("interactor error")

// User errors

var ErrLoginValidation = errors.Join(ErrInteractor, errors.New("login validation error"))
var ErrLoginTooLong = errors.Join(ErrLoginValidation, errors.New("login too long"))
var ErrLoginTooShort = errors.Join(ErrLoginValidation, errors.New("login too short"))
var ErrLoginChars = errors.Join(ErrLoginValidation, errors.New("login contains incorrect chars"))
var ErrLoginTaken = errors.Join(ErrInteractor, errors.New("login already taken error"))
var ErrWrongCredentials = errors.Join(ErrInteractor, errors.New("wrong credentials"))
var ErrPasswordComplexity = errors.Join(ErrInteractor, errors.New("password complexity error"))
var ErrUserNotFound = errors.Join(ErrInteractor, errors.New("user not found"))
var ErrUserKeysNotFound = errors.Join(ErrInteractor, errors.New("user keys not found"))

// Data errors

var ErrDataInfoNotFound = errors.Join(ErrInteractor, errors.New("data info not found"))
var ErrDataConflict = errors.Join(ErrInteractor, errors.New("data conflict error"))
var ErrDataDeleted = errors.Join(ErrInteractor, errors.New("data deleted error"))

// Gateway errors

var ErrGateway = errors.New("gateway error")
var ErrOffline = errors.Join(ErrGateway, errors.New("offline"))
