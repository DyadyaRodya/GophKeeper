package models

import "time"

// DataInfo meta info for stored data
type DataInfo struct {
	UUID        string
	OwnerUUID   string
	LastUpdated time.Time
	IsDeleted   bool
}

// DataType type of enum
type DataType string

const (
	CredsDataType  DataType = "creds"
	CardDataType   DataType = "card"
	TextDataType   DataType = "text"
	BinaryDataType DataType = "binary"
)

// CredsData struct with login/password pair
type CredsData struct {
	Login string `json:"login"`
	Pass  string `json:"pass"`
}

// CardData struct with card data
type CardData struct {
	Number string `json:"number"`
	Date   string `json:"date"`
	Code   string `json:"code"`
	Holder string `json:"holder"`
}

// TextData type for text data
type TextData struct {
	Data string `json:"data"`
}

// BinaryData type for binary data
type BinaryData struct {
	Data []byte `json:"data"`
}

// RawData root struct for data
type RawData struct {
	DataType DataType `json:"type"`
	MetaInfo string   `json:"metaInfo"`

	Creds  *CredsData  `json:"creds"`
	Card   *CardData   `json:"card"`
	Text   *TextData   `json:"text"`
	Binary *BinaryData `json:"binary"`
}
