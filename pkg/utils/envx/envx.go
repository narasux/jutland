package envx

import "os"

// Get 读取环境变量，支持默认值
func Get(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
