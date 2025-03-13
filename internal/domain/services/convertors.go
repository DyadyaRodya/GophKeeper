package services

import (
	"bytes"
	"encoding/json"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

// DataConvertor service for converting RawData to bytes array and bytes array to RawData
type DataConvertor struct {
}

// NewDataConvertor constructor for DataConvertor
func NewDataConvertor() *DataConvertor {
	return &DataConvertor{}
}

// ConvertRawDataToBytes converts given RawData to bytes array
func (d *DataConvertor) ConvertRawDataToBytes(rawData *domainmodels.RawData) ([]byte, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(rawData)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ConvertBytesToRawData converts given bytes array to RawData
func (d *DataConvertor) ConvertBytesToRawData(dataBytes []byte) (*domainmodels.RawData, error) {
	buf := bytes.NewReader(dataBytes)
	rawData := &domainmodels.RawData{}
	err := json.NewDecoder(buf).Decode(rawData)
	if err != nil {
		return nil, err
	}
	return rawData, nil
}
