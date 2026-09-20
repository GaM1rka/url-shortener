package service

type CodeGenerator interface {
	Generate() (string, error)
}

type RandomGenerator struct{}

func NewRandomGenerator() *RandomGenerator {
	return &RandomGenerator{}
}

func (g *RandomGenerator) Generate() (string, error) {
	return "", nil
}