package pathx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/utils/pathx"
)

func TestGetCurPKGPath(t *testing.T) {
	// 该函数返回结果与调用位置相关，在这里是结果是 .../pkg/util/path
	assert.True(t, strings.HasSuffix(pathx.GetCurPKGPath(), "pkg/utils/pathx"))
}
