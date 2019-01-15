package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

func generateServerKey() *rsa.PrivateKey {
	log("Generating Server Key")

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	handleError("Failed to generate a server key: ", err)

	pemPrivateFile, err := os.Create("private_key.pem")
	defer pemPrivateFile.Close()
	handleError("Failed to create private_key.pem: ", err)
	var pemPrivateBlock = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
	}
	err = pem.Encode(pemPrivateFile, pemPrivateBlock)
	handleError("Failed to encode private key to pem format ", err)

	pemPublicFile, err := os.Create("public_key.pem")
	defer pemPrivateFile.Close()
	handleError("Failed to create public_key.pem: ", err)
	var pemPublicBlock = &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&serverKey.PublicKey),
	}
	err = pem.Encode(pemPublicFile, pemPublicBlock)
	handleError("Failed to encode public key to pem format ", err)

	return serverKey
}

func loadServerKey() *rsa.PrivateKey {
	log("Loading Server Key")

	privateKeyFile, err := os.Open("private_key.pem")
	handleError("Failed to open private_key.pem: ", err)

	pemfileinfo, _ := privateKeyFile.Stat()
	size := pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	defer privateKeyFile.Close()
	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	handleError("Failed to open private_key.pem: ", err)

	return privateKeyImported
}
