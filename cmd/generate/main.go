//go:generate go run main.go

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
	"tunnel/cert"
	"tunnel/core"
)

func main() {
	// 生成私钥
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println("Failed to generate private key:", err)
		return
	}

	// 设置证书的基本信息
	notBefore := time.Now()
	notAfter := notBefore.Add(100 * 365 * 24 * time.Hour) // 100年

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		fmt.Println("Failed to generate serial number:", err)
		return
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{cert.ServerName},
	}

	// 生成自签名证书
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		fmt.Println("Failed to create certificate:", err)
		return
	}

	// 将证书保存到文件
	certOut, err := os.Create("../../cert/cert.pem")
	if err != nil {
		fmt.Println("Failed to open cert.pem for writing:", err)
		return
	}
	defer core.CloseAndLog(certOut)

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		fmt.Println("Failed to write data to cert.pem:", err)
		return
	}

	// 将私钥保存到文件
	keyOut, err := os.Create("../../cert/key.pem")
	if err != nil {
		fmt.Println("Failed to open key.pem for writing:", err)
		return
	}
	defer core.CloseAndLog(keyOut)

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		fmt.Println("Failed to marshal private key:", err)
		return
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		fmt.Println("Failed to write data to key.pem:", err)
		return
	}

	fmt.Println("Certificate and key have been successfully generated and saved.")
}
