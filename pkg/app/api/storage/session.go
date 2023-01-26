package storage

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"soldr/pkg/system"
	"soldr/pkg/version"
)

// MakeCookieStoreKey is function to generate secure key for cookie store
func MakeCookieStoreKey() []byte {
	md5Hash := func(value, salt string) string {
		hash := md5.Sum([]byte(value + "|" + salt))
		return hex.EncodeToString(hash[:])
	}
	key := strings.Join([]string{
		md5Hash(version.GetBinaryVersion(), "972bf553c89cd103feb198f62a24e305b06a8840"),
		system.MakeAgentID(),
	}, "|")
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}
