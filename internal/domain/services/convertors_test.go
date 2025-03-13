package services

import (
	"reflect"
	"testing"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

func TestDataConvertor(t *testing.T) {
	convertor := NewDataConvertor()

	tests := []*domainmodels.RawData{
		{
			DataType: "creds",
			MetaInfo: "creds",
			Creds: &domainmodels.CredsData{
				Login: "login",
				Pass:  "pass",
			},
			Card:   nil,
			Text:   nil,
			Binary: nil,
		},
		{
			DataType: "card",
			MetaInfo: "card",
			Creds:    nil,
			Card: &domainmodels.CardData{
				Number: "1234567891011121",
				Date:   "09/01",
				Code:   "123",
				Holder: "JOHN DOE",
			},
			Text:   nil,
			Binary: nil,
		},
		{
			DataType: "text",
			MetaInfo: "text",
			Creds:    nil,
			Card:     nil,
			Text:     &domainmodels.TextData{Data: "text"},
			Binary:   nil,
		},
		{
			DataType: "binary",
			MetaInfo: "binary",
			Creds:    nil,
			Card:     nil,
			Text:     nil,
			Binary:   &domainmodels.BinaryData{Data: []byte("binary")},
		},
	}
	for _, test := range tests {
		t.Run(string(test.DataType), func(t *testing.T) {
			bytesData, err := convertor.ConvertRawDataToBytes(test)
			if err != nil {
				t.Fatal(err)
			}

			rawData, err := convertor.ConvertBytesToRawData(bytesData)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(test, rawData) {
				t.Errorf("expected: %v, got: %v", test, rawData)
			}
		})
	}
}
