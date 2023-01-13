package types

type LTAC struct {
	LTACPublic
	Key []byte
}

type LTACPublic struct {
	Cert []byte
	CA   []byte
}

type InitConnectionPack struct {
	LTAC   *LTAC
	SSALua []byte
	SBH    []byte
}
