// Package crpt ...
package crpt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"

	"github.com/kissanjamgit/privatestream/config"
)

func DecryptAESBase64URLSafe(Config *config.Config, input string) (value string, err error) {
	cipherByte, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return
	}
	// res, err := http.Get(Config.SecretKeyURI)
	// if err != nil {
	// 	return
	// }
	// if res.StatusCode != http.StatusOK {
	// 	err = fmt.Errorf(`res.StatusCode != http.StatusOK`)
	// 	return
	// }
	// key, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	return
	// }
	block, err := aes.NewCipher(Config.SecretKey)
	if err != nil {
		return
	}
	mode := cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize))
	valueByte := make([]byte, len(cipherByte))
	mode.CryptBlocks(valueByte, cipherByte)
	func() {
		padNo := valueByte[len(valueByte)-1]
		valueByte = valueByte[:len(valueByte)-int(padNo)]
	}()
	value = string(valueByte)
	return
}

func EncryptAESBase64URLSafe(Config *config.Config, input []byte) (value string, err error) {
	func() {
		padNeeded := aes.BlockSize - len(input)%aes.BlockSize
		pad := bytes.Repeat([]byte{byte(padNeeded)}, padNeeded)
		input = append(input, pad...)
	}()
	// res, err := http.Get(Config.SecretKeyURI)
	// if err != nil {
	// 	return
	// }
	// if res.StatusCode != http.StatusOK {
	// 	err = fmt.Errorf(`res.StatusCode != http.StatusOK`)
	// 	return
	// }
	// key, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	return
	// }
	block, err := aes.NewCipher(Config.SecretKey)
	if err != nil {
		return
	}
	mode := cipher.NewCBCEncrypter(block, make([]byte, aes.BlockSize))
	cipherByte := make([]byte, len(input))
	mode.CryptBlocks(cipherByte, input)

	value = base64.RawURLEncoding.EncodeToString(cipherByte)
	return
}
