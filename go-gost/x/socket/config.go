package socket

import (
	"os"
	"sync"

	"github.com/go-gost/x/config"
)

// configMutex 保护配置文件的并发写入
var configMutex sync.Mutex

func saveConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()

	file := "gost.json"

	f, err := os.Create(file)
	if err != nil {
		return
	}
	defer f.Close()

	if err := config.Global().Write(f, "json"); err != nil {

		return
	}

	return
}
