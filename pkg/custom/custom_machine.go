package custom

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/majd/ipatool/v2/pkg/util/machine"
)

type customachine struct {
	machine.Machine
	email string
}

func md5String(s string) string {
	sum := md5.Sum([]byte(s))         // [16]byte
	return hex.EncodeToString(sum[:]) // 32 chars hex
}

func NewMachine(email string, args machine.Args) machine.Machine {
	return &customachine{
		machine.New(args), email,
	}
}

func (m *customachine) MacAddress() (string, error) {
	md5Hex := md5String(m.email + "teniux")
	guid := md5Hex[:12] // 中间12位

	return guid, nil
}

func (m *customachine) HomeDirectory() string {
	baseDir, err := os.Getwd() // 当前工作目录（常见理解的“当前应用程序目录”）
	if err != nil {
		baseDir = "."
	}

	dirName := strings.ToLower(m.email)
	first := dirName[:1]

	dir := filepath.Join(baseDir, "UserData", first, dirName)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return baseDir
	}

	return dir
}
