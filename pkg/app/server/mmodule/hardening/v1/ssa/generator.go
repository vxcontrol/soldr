package ssa

type NOPSSAGenerator struct{}

func NewNOPSSAGenerator() *NOPSSAGenerator {
	return &NOPSSAGenerator{}
}

func (g *NOPSSAGenerator) GenerateSSAScript(scriptEncodingKey []byte) ([]byte, error) {
	return []byte(""), nil
}
