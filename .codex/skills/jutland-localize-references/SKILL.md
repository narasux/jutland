---
name: jutland-localize-references
description: Localize and maintain Jutland encyclopedia reference data across configs/references.json5 and configs/references.LOCALE.json5. Use when translating ship or aircraft names, types, armament summaries, descriptions, authors, or source links; adding another reference locale; fixing runtime language switching or mixed-language wrapping; or validating localized reference parity and attribution.
---

# Jutland 图鉴资料本地化

## 核心规则

- 在仓库根目录工作。先阅读 `AGENTS.md` 并执行 `git status --short`，保留用户已有改动和素材。
- 所有项目路径以仓库根目录为基准；命令和辅助脚本不得写死机器绝对路径。
- 把 `configs/references.json5` 视为简体中文基准，把 `configs/references.<locale>.json5` 视为同结构的完整本地化文件。
- 保持所有语言的顶层 `name` 集合完全一致。`name` 是稳定内部 ID，不翻译、不修正大小写。
- 保持武装条目数量、顺序、数量、口径、型号和游戏数值不变；只翻译展示文本。
- 保留真实素材作者的原始署名和素材链接。只把“未知”本地化为目标语言。
- 不向第三方翻译服务发送仓库文本，除非用户明确授权。历史资料优先从公开英文来源核验。

## 关键文件与数据流

- 中文数据：`configs/references.json5`
- 英文数据：`configs/references.en.json5`
- 数据结构、按语言映射与回退：`pkg/mission/object/reference/`
- 启动期加载：`pkg/mission/object/initialize/initialize.go`
- 图鉴绘制：`pkg/collection/`
- 舰船、飞机展示名：`pkg/mission/object/unit/ship.go`、`plane.go`
- 配置说明：`configs/Readme.md`

运行时必须同时加载全部正式语言。不要在包初始化阶段只按当前设置加载一个文件：Go 包 `init` 早于 `main` 中的 `config.LoadGameSettings()`，而且设置页支持不重启切换语言。

## 工作流程

1. 盘点变更范围。
   - 统计基准文件顶层对象、武装和链接数量。
   - 搜索 `GetReference` 的所有调用方，确认本地化会影响图鉴、舰名、飞机名和建造界面。
   - 新增语言时先确认 `i18n.Language` 已正式启用，再新增对应 references 文件。

2. 维护运行时选择与回退。
   - 在 `reference` 包按 `i18n.Language` 保存独立映射。
   - `GetReference(name)` 每次依据 `i18n.CurrentLanguage()` 选择数据，缺失时回退 `zh-Hans`。
   - 加载 slice 时用索引取地址，例如 `ref := &references[idx]`；不要保存 `for range` 临时变量地址。
   - 把纯 JSON5 加载和一致性校验放在不依赖 Ebiten 的 `reference` 包。初始化包只负责加载失败时 `log.Fatal` 和注册数据，避免数据测试触发图形环境。

3. 翻译完整展示面。
   - 翻译 `displayName`、`type`、武装 `label`、武装 `value` 中的名称、`description` 和链接标题。
   - 名称采用目标语言中的权威通行名称与型号；英文可保留必要变音符号，例如 `Yūbari`、`Shōkaku`。
   - 英文武装采用稳定术语，如 `Main Battery`、`Dual-purpose Guns`、`Anti-aircraft Guns`、`Torpedoes`。
   - 英文描述依据来源重写为 2–4 句、45–70 词。覆盖身份、设计特点和主要结局，不扩写成超出卡片容量的长文。
   - 13 个飞机或通用类型若基准描述为空，保持为空；不要为了字段非空编造历史内容。

4. 审核来源与署名。
   - 历史资料优先对象对应的英文维基页面；没有单舰页面时使用舰级、型号或最近的英文概念页面。
   - 用 Wikipedia API 或实际打开页面确认链接存在或可重定向，不凭标题猜 URL。
   - DeviantArt、Shipbucket、Wikimedia Commons、Bilibili 等原创素材链接保持原 URL。
   - 来源卡片当前最多显示前两条链接。原创设计优先排列“素材署名链接”，第二条排列“英文历史/型号来源”，避免可靠来源被隐藏。

5. 处理布局与混合语言。
   - 翻译后检查卡片标题、历史正文、作者、链接、武装和 `Total Power` 区域，不只检查配置解析。
   - 文本换行应按实际内容和字体判断，不能只依据 `CurrentLanguage()`：英文界面仍可能显示中文作者名或回退文本。
   - 对 CJK 与拉丁混合文本使用 Unicode 内容检测或实际字宽折行，避免把 `5500` 这类空格分隔片段单独留在一行。
   - 长链接按卡片内宽换行，并为每行维护同一 URL 的点击区域。

6. 更新文档并验证。
   - 在 `configs/Readme.md` 记录语言文件、稳定字段和回退行为。
   - 运行 `gofmt` 处理 Go 文件，执行 `git diff --check`。
   - 运行 `go test ./pkg/mission/object/reference`，覆盖解析、重复 ID、语言集合、武装数量、URL、运行时选择和中文回退。
   - 运行受影响包测试和 `go build ./...`。
   - `go test ./...` 在无 GUI 的 macOS 环境可能因 Ebiten 初始化失败；先保证纯数据测试不依赖 Ebiten，并如实报告全量测试的实际失败原因。

## 完成标准

- 每个正式语言文件都能解析，且对象集合与武装条目数量一致。
- 目标语言的展示字段无意外中文残留；真实作者原文属于允许项。
- 每个英文对象至少有一个可访问的英文历史、舰级或型号来源；所有素材署名链接仍可追溯。
- 切换语言后，舰名、机名、图鉴资料和武装摘要立即更新；缺失本地化时稳定回退中文。
- 目标测试和完整构建实际通过，图鉴在常用分辨率下没有截断、越界或异常孤行。
