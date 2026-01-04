package custom

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/majd/ipatool/v2/pkg/util/machine"
)

type customachine struct {
	machine.Machine
	email string
}

func NewMachine(email string, args machine.Args) machine.Machine {
	return &customachine{
		machine.New(args), email,
	}
}

func (*customachine) MacAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	if len(interfaces) == 0 {
		return "", fmt.Errorf("could not find network interfaces: %w", err)
	}

	for _, netInterface := range interfaces {
		addr := netInterface.HardwareAddr.String()
		if addr != "" {
			return addr, nil
		}
	}

	return "", fmt.Errorf("could not find network interfaces with a valid mac address: %w", err)
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
