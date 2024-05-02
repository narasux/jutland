package envx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/utils/envx"
)

func TestGetEnvWithDefault(t *testing.T) {
	// 不存在的环境变量
	ret := envx.Get("NOT_EXISTS_ENV_KEY", "ENV_VAL")
	assert.Equal(t, "ENV_VAL", ret)

	// 已存在的环境变量
	ret = envx.Get("PATH", "")
	assert.NotEqual(t, "", ret)
}
