package interfaces

import domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"

type (
	RawDataToBytesConverter interface {
		ConvertRawDataToBytes(rawData *domainmodels.RawData) ([]byte, error)
	}
	BytesToRawDataConverter interface {
		ConvertBytesToRawData(dataBytes []byte) (*domainmodels.RawData, error)
	}
)
