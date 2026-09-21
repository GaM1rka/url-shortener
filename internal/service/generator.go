package service

import (
	"crypto/rand"
	"fmt"
)

const (
	shortCodeLength   = 10
	shortCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_"
	randomByteLimit   = 256 - 256%len(shortCodeAlphabet)
)

type CodeGenerator interface {
	Generate() (string, error)
}

type RandomGenerator struct{}

func NewRandomGenerator() *RandomGenerator {
	return &RandomGenerator{}
}

func (*RandomGenerator) Generate() (string, error) {
	code := make([]byte, shortCodeLength)
	randomBytes := make([]byte, shortCodeLength*2)
	written := 0

	for written < len(code) {
		if _, err := rand.Read(randomBytes); err != nil {
			return "", fmt.Errorf("read cryptographic random bytes: %w", err)
		}

		for _, value := range randomBytes {
			if int(value) >= randomByteLimit {
				continue
			}

			code[written] = shortCodeAlphabet[int(value)%len(shortCodeAlphabet)]
			written++
			if written == len(code) {
				break
			}
		}
	}

	return string(code), nil
}
