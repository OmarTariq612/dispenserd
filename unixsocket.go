package dispenserd

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

func UnixListen(path string) (net.Listener, error) {
	conn, err := net.Dial("unix", path)
	if err == nil {
		conn.Close()
		return nil, fmt.Errorf("%s: address already in use", path)
	}
	_ = os.Remove(path)
	perm := os.FileMode(0666)
	sockDir := filepath.Dir(path)
	if _, err := os.Stat(sockDir); os.IsNotExist(err) {
		_ = os.MkdirAll(sockDir, 0755)

		if fi, err := os.Stat(sockDir); err == nil && fi.Mode()&0077 == 0 {
			if err := os.Chmod(sockDir, 0755); err != nil {
				log.Print(err)
			}
		}
	}
	pipe, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(path, perm)
	return pipe, err
}
