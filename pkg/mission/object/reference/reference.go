package reference

type Link struct {
	// 名称
	Name string `json:"name"`
	// 链接
	URL string `json:"url"`
}

// Reference 对象引用信息
type Reference struct {
	// 名称
	Name string `json:"name"`
	// 描述
	Description []string `json:"description"`
	// 原作者
	Author string `json:"author"`
	// 链接
	Links []Link `json:"links"`
}

var referencesMap = map[string]*Reference{}

// GetReference 获取对象引用
func GetReference(name string) *Reference {
	return referencesMap[name]
}

// SetReference 设置对象引用（供初始化使用）
func SetReference(name string, ref *Reference) {
	referencesMap[name] = ref
}
