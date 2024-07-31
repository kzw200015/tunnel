package core

import "crypto/sha256"

func Hash(input string) [32]byte {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashBytes := hasher.Sum(nil)
	return [32]byte(hashBytes)
}
