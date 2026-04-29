package generator

import "github.com/google/uuid"

func NewVariationSeed() string {
	return uuid.New().String()
}
