package vm

type SimplePingResponder struct{}

func NewSimplePingResponder() *SimplePingResponder {
	return &SimplePingResponder{}
}

func (r *SimplePingResponder) GeneratePingResponse(nonce []byte) ([]byte, error) {
	return nonce, nil
}
