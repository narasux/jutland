package reference

import "github.com/narasux/jutland/pkg/i18n"

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
	// 舰种名称，用于保留“重巡洋舰”“装甲航空母舰”等精细分类。
	// 缩写、排水量、速度、费用和减伤等数据统一从舰船配置读取。
	Type string `json:"type"`
	// 图鉴武装摘要
	Armaments []InfoItem `json:"armaments"`
	// 描述
	Description string `json:"description"`
	// 原作者
	Author string `json:"author"`
	// 链接
	Links []Link `json:"links"`
}

var referencesByLanguage = map[i18n.Language]map[string]*Reference{}

// GetReference 获取对象引用
func GetReference(name string) *Reference {
	lang := i18n.CurrentLanguage()
	if references := referencesByLanguage[lang]; references != nil {
		if ref := references[name]; ref != nil {
			return ref
		}
	}
	return referencesByLanguage[i18n.LanguageZhHans][name]
}

// SetReference 设置对象引用（供初始化使用）
func SetReference(lang i18n.Language, name string, ref *Reference) {
	if referencesByLanguage[lang] == nil {
		referencesByLanguage[lang] = map[string]*Reference{}
	}
	referencesByLanguage[lang][name] = ref
}
