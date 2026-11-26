package tutorial

// ActionID 配置中的语义化输入动作。
type ActionID string

const (
	ActionSelectArea  ActionID = "select-area"
	ActionMove        ActionID = "move"
	ActionGroup       ActionID = "group"
	ActionSelectGroup ActionID = "select-group"
	ActionScatter     ActionID = "scatter"
	ActionDive        ActionID = "dive"
	ActionSonar       ActionID = "sonar"
	ActionMap         ActionID = "map"
	ActionPause       ActionID = "pause"
	ActionResume      ActionID = "resume"
	ActionReinforce   ActionID = "reinforce"
	ActionNudge       ActionID = "nudge"
	ActionScout       ActionID = "scout"
	ActionZoom        ActionID = "zoom"
	ActionTerminal    ActionID = "terminal"
	ActionSidebar     ActionID = "sidebar"
	ActionSummon      ActionID = "summon"
	ActionCancel      ActionID = "cancel"
	ActionBack        ActionID = "back"
)

// TokenKind 键帽/鼠标徽标的类型。
type TokenKind int

const (
	// TokenKey 普通键盘按键。
	TokenKey TokenKind = iota
	// TokenMouseLeft 鼠标左键。
	TokenMouseLeft
	// TokenMouseRight 鼠标右键。
	TokenMouseRight
	// TokenMouseDrag 按住拖动。
	TokenMouseDrag
	// TokenMouseWheel 鼠标滚轮。
	TokenMouseWheel
)

// Token 一个键帽或鼠标徽标。
type Token struct {
	Kind  TokenKind
	Label string
}

// ActionSpec 一个语义动作的显示信息。
type ActionSpec struct {
	Tokens []Token
	// Hold 按住前置键再执行后续动作（例如 Ctrl + 数字）。
	Hold bool
}

var actionSpecs = map[ActionID]ActionSpec{
	ActionSelectArea: {
		Tokens: []Token{{Kind: TokenMouseLeft}, {Kind: TokenMouseDrag}},
	},
	ActionMove: {
		Tokens: []Token{{Kind: TokenMouseRight}},
	},
	ActionGroup: {
		Tokens: []Token{{Kind: TokenKey, Label: "Ctrl"}, {Kind: TokenKey, Label: "1-0"}},
		Hold:   true,
	},
	ActionSelectGroup: {
		Tokens: []Token{{Kind: TokenKey, Label: "0-9"}},
	},
	ActionScatter: {
		Tokens: []Token{{Kind: TokenKey, Label: "X"}},
	},
	ActionDive: {
		Tokens: []Token{{Kind: TokenKey, Label: "D"}},
	},
	ActionSonar: {
		Tokens: []Token{{Kind: TokenKey, Label: "V"}},
		Hold:   true,
	},
	ActionMap: {
		Tokens: []Token{{Kind: TokenKey, Label: "M"}},
	},
	ActionPause: {
		Tokens: []Token{{Kind: TokenKey, Label: "Esc"}},
	},
	ActionResume: {
		Tokens: []Token{{Kind: TokenKey, Label: "Enter"}},
	},
	ActionReinforce: {
		Tokens: []Token{{Kind: TokenKey, Label: "B"}},
	},
	ActionNudge: {
		Tokens: []Token{{Kind: TokenKey, Label: "↑ ↓ ← →"}},
	},
	ActionScout: {
		Tokens: []Token{{Kind: TokenKey, Label: "Shift"}, {Kind: TokenMouseRight}},
		Hold:   true,
	},
	ActionZoom: {
		Tokens: []Token{{Kind: TokenMouseWheel}},
	},
	ActionTerminal: {
		Tokens: []Token{
			{Kind: TokenKey, Label: "Ctrl"},
			{Kind: TokenKey, Label: "Shift"},
			{Kind: TokenKey, Label: "`"},
		},
		Hold: true,
	},
	ActionSidebar: {
		Tokens: []Token{{Kind: TokenMouseLeft}},
	},
	ActionSummon: {
		Tokens: []Token{{Kind: TokenKey, Label: "Enter"}},
	},
	ActionCancel: {
		Tokens: []Token{{Kind: TokenKey, Label: "Backspace"}},
	},
	ActionBack: {
		Tokens: []Token{{Kind: TokenKey, Label: "Esc"}},
	},
}

// TokensFor 返回语义动作的显示键帽。
func TokensFor(action ActionID) (ActionSpec, bool) {
	spec, ok := actionSpecs[action]
	return spec, ok
}

// ActionValid 判断配置里的动作 ID 是否存在。
func ActionValid(action ActionID) bool {
	_, ok := actionSpecs[action]
	return ok
}
