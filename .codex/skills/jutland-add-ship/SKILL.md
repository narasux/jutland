---
name: jutland-add-ship
description: Add or update a Jutland ship from user-confirmed transparent source images and historical references. Use after white-background cleanup has been reviewed; handles view extraction, orientation, aspect-ratio-preserving scaling, image backups, ship and weapon configuration, references, hybrid ships, TestAll integration, and validation. Do not use this skill to clean white backgrounds.
---

# Jutland 添加战舰

## 核心规则

- 在仓库根目录阅读 `AGENTS.md`，执行 `git status --short`，保留无关用户改动。
- 所有项目路径以仓库根目录为基准；脚本不得写死机器绝对路径，外部输入和输出路径由调用者显式传入。
- 只接受已完成白底清理且经用户确认的透明图片。若输入仍有白底或未确认，先使用 `jutland-clean-ship-image` 并在预览后停止。
- 本 skill 不执行白底透明化。只允许拆分、裁剪、旋转和等比例缩放素材，不重绘或风格化。
- 保持 `name`、PNG 文件名及所有配置引用一致；优先复用现有舰船、武器、飞机和发射器模式。
- 不为一次性需求扩展数据模型或通用玩法。默认只做构建验证；修改 Go 行为时再运行相关测试。

## 目标文件

- 舰船：`configs/ships.json5`
- 武器：`configs/guns.json5` 及其弹药/发射器配置
- 图鉴：`configs/references.json5`
- 测试任务：`configs/missions.json5` 中的 `TestAll`
- 正式图片：`resources/images/ships/{top,side}/<type>/<name>.png`
- 缩放前原尺寸图：放项目根目录，命名为 `<ship name> side.png`、`<ship name> top.png`，例如 `princeton side.png`、`princeton top.png`
- 缩放脚本：`scripts/resize_ship_image.py`

`type` 必须匹配 `pkg/resources/images/ship/ship.go` 扫描的现有目录，例如 `aircraft_carrier`、`battleship`、`cruiser`、`destroyer`、`frigate`、`torpedo_boat`、`cargo`、`hospital`。

## 工作流程

### 1. 确认输入与现有模式

- 确定 `name`、展示名、国家、`type`、`typeAbbr`、资料链接、素材来源和作者。常用舰名转换为 snake_case；高风险命名歧义在写文件前确认。
- 阅读相邻的舰船、图鉴和 `TestAll` 条目，选择 1~3 艘同阵营、同年代、同定位舰船作为费用、减伤、加速度和转向基准。
- 优先读取用户提供的资料链接；无法访问时索要长度、舰体宽度、排水量、速度和武装等必要数据。

### 2. 拆分并定向图片

- 从用户确认的透明图中提取需要的视图；默认只保留俯视和侧视，除非用户另有要求。
- 边界不明显时按 alpha 或非背景像素做行列密度扫描。裁剪必须保留桅杆、天线、索具和舷外结构；侧视图最上方内容上方至少保留 50 px 原始分辨率余量。
- 裁剪后确认最外层非透明像素不贴边，但舰艏、舰艉只保留防止截断所需的小安全边，通常为原图 2~6 px，且不得超过舰体长轴的 1%。不要把整张素材画布或大段透明区域带入缩放；否则按真实长度生成的资源会让可见舰体偏小。
- 以主视图的 alpha 边界核对舰艏、舰艉裁切位置，忽略签名、网址、比例尺和其他视图的零散像素。若原图某端已经贴边，在该端补 2~4 px 透明安全边，不得继续裁掉船体。
- 按现有资源方向统一：俯视图舰艏朝上，侧视图舰艏朝右。只旋转或裁剪，不改变舰体比例。
- 若此时发现白底残留，返回 `jutland-clean-ship-image` 处理并等待用户重新确认，不在本 skill 内清理。

### 3. 备份并等比例缩放

- 必须使用 `scripts/resize_ship_image.py`，不要为单舰临时重写缩放代码。系统 Python 缺少 Pillow 时，先调用 `load_workspace_dependencies` 获取工作区 Python。
- 脚本会在缩放前校验并保留原尺寸图；已有原尺寸图内容不同时会拒绝覆盖。备份路径使用项目根目录文件，并用实际舰名替代 `<ship name>`，例如 `princeton side.png`、`princeton top.png`。不要使用 `resources/images/ships/original`。
- `length` 和 `width` 写入四舍五入后的真实舰体尺寸；航母 `width` 使用舰体宽度，不用飞行甲板外廓宽度。
- 正式 PNG 是运行时的 `zoom=4` 资源，加载器再生成 1/2 和 1/4 档；因此长轴通常必须保持约 `length*4`。提高该数值会改变舰船显示尺寸，不能用作保细节手段。
- 在拆分阶段去除不必要的透明空白，尤其是决定实际显示长度的舰艏、舰艉方向；侧视图仍保留桅杆上方至少 50 px 原始分辨率余量。缩放前记录画布边界与 alpha 边界的间距，确认长轴两端只剩小安全边。
- 使用 `--target-long-axis` 或单个目标轴，另一轴由脚本按原比例计算。默认 `--mode line-art` 在预乘 alpha 的 Lanczos 缩放后做固定的温和锐化；`--mode plain` 只做 Lanczos。
- 缩放比例 `<0.5` 时分别生成 `plain` 和 `line-art` 候选并比较高对比预览。锐化只能增强线条对比，无法恢复缩放后不足 1 px 的结构；必要时换用更高分辨率源图。
- 始终使用 `--preview-dir` 生成黑底和洋红底预览，并用 `view_image` 检查。若仍有白底残留，交回 `jutland-clean-ship-image` 处理并确认。
- 用 `sips` 确认目标尺寸，检查脚本报告的缩放率和细节风险，再使用 `--overwrite-output` 覆盖正式图片。

示例：

```bash
python3 .codex/skills/jutland-add-ship/scripts/resize_ship_image.py \
  input.png output.png \
  --backup "example top.png" \
  --target-long-axis 1044 --mode line-art --preview-dir "$(mktemp -d)"
```

### 4. 添加舰船配置

- 将条目放到同舰种、同阵营附近。
- `totalHP` 优先使用满载排水量；`maxSpeed` 使用节。费用、时间、减伤、加速度和转向从选定基准推导。
- 按资料与图片配置主炮、副炮、防空炮、鱼雷、火箭和舰载机挂点。只引用存在的配置名称，并核对 `posPercent`、射界和数量。
- 航空母舰只编入仓库已有飞机；缺少历史机型时复用同阵营、同任务角色的可用机型。图鉴直接列出实际配置，不添加模板化免责声明。

#### Mark 专用武器

- 现有通用武器不能表达明确且有性能差异的 Mark 型号时，新增独立条目；不要修改已有舰船引用。
- 命名使用 `country/caliber/length/barrel_count/MKtype`，例如 `US/203/55/3/MK16`；无 Mark 或单装时沿用仓库现有简写。
- 弹药可复用最接近的现有条目；按历史射速、射程和散布调整性能，`bulletCount` 表示每座炮的炮管数。
- 口径 `>=40mm` 的专用防空炮保留 Mark 标识；更小口径沿用通用条目。同步更新图鉴武装描述。

### 5. 处理混合舰种

- 航空战列舰、航空巡洋舰按主要水面身份选择现有 `type`，航空能力通过 `aircraft` 表达。
- 只有独立 UI 分类、资源路由或玩法规则确实需要时，才扩展 Go 枚举和加载器。
- 图片保留甲板、弹射器和舷外平台的原始比例；配置 `width` 仍使用舰体宽度。
- 不为单舰修改全局起降、返航或回收逻辑。`TestAll` 只按主 `type` 分组一次。

### 6. 更新图鉴与 TestAll

- 在所有正式语言的 references 文件中添加同名条目：规格、游戏中实际武装、简短历史、作者，以及资料链接。
- 每个舰船图鉴的 `links` 统一只保留两项且顺序固定：第一项是用户提供或原作者发布的素材来源；第二项是目标语言维基百科中与该舰、舰级或型号最直接相关的条目。目标语言没有对应条目时，第二项回退到英文维基百科。
- 用 Wikipedia API、搜索结果或实际打开页面确认维基链接存在或可重定向；不得凭标题猜测 URL。四种语言允许因本地化维基条目不同而使用不同的第二项 URL，但第一项素材来源必须保持一致。
- 在 `TestAll` 对应 `providedShipNames` 添加舰名，并在同舰种网格中增加一艘己方 `HA` 初始舰，避免重叠。
- 除非用户明确要求，不修改其他任务。

### 7. 验证

- 确认原尺寸备份和正式俯视/侧视 PNG 均存在，方向正确，缩放保持长宽比；正式图舰艏、舰艉不得有会使可见舰体明显小于 `length*4` 的透明留白。
- 搜索舰名，确认 `ships.json5`、`references.json5`、`TestAll` 和 PNG 文件名一致且没有重复定义。
- 搜索所有武器、弹药、发射器和飞机引用，确认上游配置存在。
- 用 `view_image` 检查正式 PNG 的完整轮廓和透明边缘；用 `git diff --check` 检查文本问题。
- 运行 `go build ./pkg/...`。最终说明资料来源、配置选择和未验证的视觉区域。
