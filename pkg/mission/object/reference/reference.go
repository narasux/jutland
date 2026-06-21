package reference

// Link 链接信息
type Link struct {
	// 名称
	Name string `json:"name"`
	// 链接
	URL string `json:"url"`
}

// InfoItem 图鉴展示信息项
type InfoItem struct {
	// 标签
	Label string `json:"label"`
	// 内容
	Value string `json:"value"`
}

// Reference 对象引用信息
type Reference struct {
	// 名称
	Name string `json:"name"`
	// 展示用名称
	DisplayName string `json:"displayName"`
	// 图鉴核心参数
	Specs []InfoItem `json:"specs"`
	// 图鉴武装摘要
	Armaments []InfoItem `json:"armaments"`
	// 描述
	Description string `json:"description"`
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
