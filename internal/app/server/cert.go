package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"

	"go.uber.org/zap"
)

func ensureCertAndKeyExist(logger *zap.Logger) {
	var certPEM bytes.Buffer
	var privateKeyPEM bytes.Buffer

	// try to read cert file
	certFile, err := os.OpenFile("goshortener.cert.pem", os.O_RDONLY, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Fatal("os.OpenFile for goshortener.cert.pem", zap.Error(err))
		}
	} else {
		defer certFile.Close()

		var n int64
		n, err = certPEM.ReadFrom(certFile)
		if err != nil || n == 0 {
			logger.Fatal("certPEM.ReadFrom", zap.Error(err), zap.Int64("n", n))
		}
	}

	// try to read private key file
	keyFile, err := os.OpenFile("goshortener.key.pem", os.O_RDONLY, 0644)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Fatal("os.OpenFile for goshortener.cert.pem", zap.Error(err))
		}
	} else {
		defer keyFile.Close()

		var n int64
		n, err = privateKeyPEM.ReadFrom(keyFile)
		if err != nil || n == 0 {
			logger.Fatal("privateKeyPEM.ReadFrom", zap.Error(err), zap.Int64("n", n))
		}
	}

	if certPEM.Len() == 0 || privateKeyPEM.Len() == 0 {
		certPEM.Reset()       // if one of the files was not empty
		privateKeyPEM.Reset() // if one of the files was not empty

		cert := &x509.Certificate{
			SerialNumber: big.NewInt(1658),
			Subject: pkix.Name{
				Organization: []string{"Go.Shortener"},
				Country:      []string{"RU"},
			},
			IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().AddDate(10, 0, 0),
			SubjectKeyId: []byte{1, 2, 3, 4, 6},
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature,
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			logger.Fatal("rsa.GenerateKey error", zap.Error(err))
		}

		certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
		if err != nil {
			logger.Fatal("x509.CreateCertificate error", zap.Error(err))
		}

		err = pem.Encode(&certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		})
		if err != nil {
			logger.Fatal("pem.Encode", zap.Error(err))
		}

		err = pem.Encode(&privateKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
		if err != nil {
			logger.Fatal("pem.Encode", zap.Error(err))
		}

		// save new values to files
		err = os.WriteFile("goshortener.cert.pem", certPEM.Bytes(), 0644)
		if err != nil {
			logger.Fatal("os.WriteFile for goshortener.cert.pem", zap.Error(err))
		}
		err = os.WriteFile("goshortener.key.pem", privateKeyPEM.Bytes(), 0644)
		if err != nil {
			logger.Fatal("os.WriteFile for goshortener.key.pem", zap.Error(err))
		}
	}
}
