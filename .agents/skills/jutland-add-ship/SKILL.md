---
name: jutland-add-ship
description: Add or update a Jutland ship from user-confirmed transparent source images and historical references. Use after white-background cleanup has been reviewed; handles view extraction, orientation, aspect-ratio-preserving scaling, image backups, ship and weapon configuration, references, hybrid ships, TestAll integration, and validation. Do not use this skill to clean white backgrounds.
---

# Jutland 添加战舰

## 核心规则

- 在仓库根目录阅读 `AGENTS.md`，执行 `git status --short`，保留无关用户改动。
- 所有项目路径以仓库根目录为基准；脚本不得写死机器绝对路径，外部输入和输出路径由调用者显式传入。
- 图片来源、原尺寸备份和中间候选统一放在 `raw_resources/` 下，不要放在仓库根目录；只有正式图片写入 `resources/`。
- 只接受已完成白底清理且经用户确认的透明图片。若输入仍有白底或未确认，先使用 `jutland-clean-ship-image` 并在预览后停止。
- 本 skill 不执行白底透明化。只允许拆分、裁剪、旋转和等比例缩放素材，不重绘或风格化。
- 保持 `name`、PNG 文件名及所有配置引用一致；优先复用现有舰船、武器、飞机和发射器模式。
- 不为一次性需求扩展数据模型或通用玩法。默认只做构建验证；修改 Go 行为时再运行相关测试。

## 目标文件

- 舰船：`configs/ships.json5`（数组顺序即图鉴顺序，插入位置见「5. 添加舰船配置 → 插入位置」）
- 武器：`configs/guns.json5` 及其弹药/发射器配置
- 图鉴：`configs/references.json5`
- 测试任务：`configs/missions.json5` 中的 `TestAll`
- 正式图片：`resources/images/ships/{top,side}/<type>/<name>.png`
- 缩放前原尺寸图：放 `raw_resources/`，命名为 `<ship name> side.png`、`<ship name> top.png`，例如 `raw_resources/princeton side.png`、`raw_resources/princeton top.png`
- 缩放脚本：`scripts/resize_ship_image.py`

`type` 必须匹配 `pkg/resources/images/ship/ship.go` 扫描的现有目录，例如 `aircraft_carrier`、`battleship`、`cruiser`、`destroyer`、`frigate`、`torpedo_boat`、`cargo`、`hospital`、`repair`。

## 工作流程

### 1. 确认输入与现有模式

- 确定 `name`、展示名、国家、`type`、`typeAbbr`、资料链接、素材来源和作者。常用舰名转换为 snake_case；高风险命名歧义在写文件前确认。
- 先按「插入位置」的三层规则（国家 → 舰种 → 舰级代际）定位新舰在 `ships.json5` 中要插入的小节和代际位置，再开始处理图片。
- 阅读相邻的舰船、图鉴和 `TestAll` 条目，选择 1~3 艘同阵营、同年代、同定位舰船作为费用、减伤、加速度和转向基准。
- 优先读取用户提供的资料链接；无法访问时索要长度、舰体宽度、排水量、速度和武装等必要数据。

### 2. 拆分并定向图片

- 从用户确认的透明图中提取需要的视图；默认只保留俯视和侧视，除非用户另有要求。
- 边界不明显时按 alpha 或非背景像素做行列密度扫描。裁剪必须保留桅杆、天线、索具和舷外结构；侧视图最上方内容上方至少保留 50 px 原始分辨率余量。
- 裁剪后确认最外层非透明像素不贴边，但舰艏、舰艉只保留防止截断所需的小安全边，通常为原图 2~6 px，且不得超过舰体长轴的 1%。不要把整张素材画布或大段透明区域带入缩放；否则按真实长度生成的资源会让可见舰体偏小。
- 以主视图的 alpha 边界核对舰艏、舰艉裁切位置，忽略签名、网址、比例尺和其他视图的零散像素。若原图某端已经贴边，在该端补 2~4 px 透明安全边，不得继续裁掉船体。
- 按现有资源方向统一：俯视图舰艏朝上，侧视图舰艏朝右。只旋转或裁剪，不改变舰体比例。
- 用户提供的舰船图默认舰艏朝右，测试图不旋转；拆分出的俯视图逆时针旋转 90°（Pillow `rotate(90, expand=True)`）后写入游戏资源。只有用户明确说明该图方向不同时才按特殊方向处理。
- 若此时发现白底残留，返回 `jutland-clean-ship-image` 处理并等待用户重新确认，不在本 skill 内清理。

### 3. 备份并等比例缩放

- 必须使用 `scripts/resize_ship_image.py`，不要为单舰临时重写缩放代码。系统 Python 缺少 Pillow 时，先调用 `load_workspace_dependencies` 获取工作区 Python。
- 脚本会在缩放前校验并保留原尺寸图；已有原尺寸图内容不同时会拒绝覆盖。备份路径使用 `raw_resources/` 下的文件，并用实际舰名替代 `<ship name>`，例如 `raw_resources/princeton side.png`、`raw_resources/princeton top.png`。不要放在仓库根目录，也不要使用 `resources/images/ships/original`。
- `length` 和 `width` 写入四舍五入后的真实舰体尺寸；航母 `width` 使用舰体宽度，不用飞行甲板外廓宽度。
- 正式 PNG 是运行时的 `zoom=4` 资源，加载器再生成 1/2 和 1/4 档；因此长轴通常必须保持约 `length*4`。提高该数值会改变舰船显示尺寸，不能用作保细节手段。
- 在拆分阶段去除不必要的透明空白，尤其是决定实际显示长度的舰艏、舰艉方向；侧视图仍保留桅杆上方至少 50 px 原始分辨率余量。缩放前记录画布边界与 alpha 边界的间距，确认长轴两端只剩小安全边。
- 使用 `--target-long-axis` 或单个目标轴，另一轴由脚本按原比例计算。默认 `--mode line-art` 在预乘 alpha 的 Lanczos 缩放后做固定的温和锐化；`--mode plain` 只做 Lanczos。
- 缩放比例 `<0.5` 时分别生成 `plain` 和 `line-art` 候选并比较高对比预览。锐化只能增强线条对比，无法恢复缩放后不足 1 px 的结构；必要时换用更高分辨率源图。
- 始终使用 `--preview-dir` 生成黑底和洋红底预览，并用 `view_image` 检查。若仍有白底残留，交回 `jutland-clean-ship-image` 处理并确认。
- 用 `sips` 确认目标尺寸，检查脚本报告的缩放率和细节风险，再使用 `--overwrite-output` 覆盖正式图片。

示例：

```bash
python3 .agents/skills/jutland-add-ship/scripts/resize_ship_image.py \
  raw_resources/example.cleaned.png resources/images/ships/top/cruiser/example.png \
  --backup "raw_resources/example top.png" \
  --target-long-axis 1044 --mode line-art --preview-dir "$(mktemp -d)"
```

### 4. 浏览器确认武器位置

- 正式俯视图缩放完成后、写入 `ships.json5` 前，必须启动 `utils/turret_marker/server.py`，让用户在浏览器中确认所有武器挂点；不得仅凭侧视图或俯视图目测直接填写 `posPercent`。
- 游戏资源保持舰艏朝上；标记页使用未旋转的横向俯视图，舰艏朝右，并保持默认 `--rotate 0` 直接让用户点击。不要把游戏资源中的竖向俯视图传给标记页，也不要靠标记参数改变方向。
- 启动时用 `--types` 写明武器类型和预期数量，例如：

```bash
python3 utils/turret_marker/server.py \
  '/tmp/example top bow-right.png' \
  --bow right \
  --types 'main:5,secondary:14,aa75:8'
```

- 用户完成标记后导出“项目 JSON”；其中的 `posPercent`、`side`、数量和类型是权威输入，按原值写入配置。不得用历史资料、目测结果或预期数量覆盖用户确认的数据。
- 炮廓炮和其他舷侧武器必须按 `side` 单舷配置：左舷只让 `leftFiringArc` 有效并写 `rightFiringArc: [0, 0]`，右舷只让 `rightFiringArc` 有效并写 `leftFiringArc: [360, 360]`。不得让同一条舷侧炮同时拥有左右两段有效射界。
- 默认端口被占用时改用 `--port`，不要终止来源不明的既有服务。

### 5. 添加舰船配置

- 按下面的「插入位置」把条目插到正确的国家/舰种/舰级位置，**不要追加到文件末尾**，也不要为图省事放到同舰种块的最后。
- `totalHP` 优先使用满载排水量；`maxSpeed` 使用节。费用、时间、减伤、加速度和转向从选定基准推导。
- 按资料与图片配置主炮、副炮、防空炮、鱼雷、火箭和舰载机挂点。只引用存在的配置名称，并核对 `posPercent`、射界和数量。
- 主炮按 `posPercent` 从大到小排列；副炮和防空炮先按武器口径从大到小分组，同口径内再按 `posPercent` 从大到小排列。同一炮座组或同一区域（如“前炮塔左右”“舰桥左右”“后部平台左右”）的条目必须相邻，并用简短注释标明区域。
- 航空母舰只编入仓库已有飞机；缺少历史机型时复用同阵营、同任务角色的可用机型。图鉴直接列出实际配置，不添加模板化免责声明。

#### 插入位置（ships.json5 排序规则）

初始化按数组顺序依次 append 到 `objUnit.AllShipNames`，图鉴列表直接沿用该顺序，因此**位置错了图鉴顺序就错**。`configs/ships.json5` 固定为三层排序：

1. **国家**：中国 `cn` → 苏联 `su` → 美国 `us` → 德国 `de` → 法国 `fr` → 日本 `jp` → 英国 `uk` → 意大利 `it` → 特殊 `special`。
   该顺序与 `pkg/mission/object/unit/types.go` 的 `AvailableNations()` 必须一致；新增国家时两处同步修改。
2. **舰种**：航空母舰 `aircraft_carrier` → 战列舰 `battleship` → 重巡洋舰 → 轻巡洋舰 → 驱逐舰 `destroyer` → 护卫舰 `frigate` → 鱼雷艇 `torpedo_boat` → 其他。
   - 巡洋舰共用 `type: "cruiser"`，按 `typeAbbr` 归段：`CA`/`CB` 属重巡洋舰段，`CL`/`CLAA`/`CM`/`CGN` 属轻巡洋舰段。
   - “其他”指 `cargo`、`repair`、`hospital` 与测试用 `default`，内部按 货轮 → 维修船 → 医疗船 → 测试单位。
3. **舰级代际（逆序）**：越新锐越靠前。同一国家同一舰种内按舰级家族分段，家族之间按最新型号逆序，家族内部按代际逆序；**同一舰级的多艘舰不排序**，保持既有相对顺序。例：美国战列舰 `montana → lowa → south_dakota → … → south_carolina → connecticut → virginia → kentucky`；日本战列舰 `edo → satsuma → yamato → nagato → ise → fuso → kongo`。
   判断代际时以该舰的舰级（见相邻条目的 `// <舰名>` 注释、references 的精细舰种、历史资料）为准，**不要用 `year` 字段直接排序**：`year` 是配置年份，同级改型、纸面方案和改装舰会互相矛盾。

子段约定（除注明外，子段排在主线之后、段内同样逆序）：

- 航母：正规航母（`CV`/`CVB`）→ 轻母 `CVL` → 护航航母 `CVE` → 商船航母 `MAC` → 水上飞机母舰 `AV` → 训练航母 `IX`。
- 重巡洋舰：`CB`（大型巡洋舰）排在 `CA` 之前。
- 轻巡洋舰：段内**统一按舰级代际逆序**，不要把防空巡洋舰 `CLAA`、布雷巡洋舰 `CM`、导弹巡洋舰 `CGN` 固定排在段首或段尾。例：美国 `long_beach（CGN 1961）→ uss_worcester_1958 → fargo → charlotte（CL-154 1945 方案）→ atlanta（1941）→ st_louis（1939）→ brooklyn（1937）`。
- 战列巡洋舰 `BC`（如 `lexington`、`kongo`、`hood`、`tiger`、`scharnhorst`）没有独立舰种，按设计年代插进该国战列舰线内。
- 实操方法：先读该「国家 × 舰种」小节**现有条目的先后**（多数小节按代际由旧到新书写），再决定新舰插在哪两艘之间；不要按 `typeAbbr` 另立子段规则，除非该小节原本就这么分段（如航母的 CVL/CVE/MAC/AV 子段）。

分组横幅注释必须同步：

- 每个「国家 × 舰种」小节以 `// <国家>海军` + `// --------- <舰种> ----------` 开头，子段写作 `// --------- 航空母舰 · 轻母（CVL） ----------`。
- 目标小节已存在时，直接在该小节内的代际位置插入条目，并保留该舰自己的 `// <舰名>` 注释；不要新建重复横幅。
- 确实需要新建国家或舰种小节时，横幅位置同样遵循上面的三层顺序。

#### Mark 专用武器

- 现有通用武器不能表达明确且有性能差异的 Mark 型号时，新增独立条目；不要修改已有舰船引用。
- 命名使用 `country/caliber/length/barrel_count/MKtype`，例如 `US/203/55/3/MK16`；无 Mark 或单装时沿用仓库现有简写。
- 弹药可复用最接近的现有条目；按历史射速、射程和散布调整性能，`bulletCount` 表示每座炮的炮管数。
- 口径 `>=40mm` 的专用防空炮保留 Mark 标识；更小口径沿用通用条目。同步更新图鉴武装描述。

### 6. 处理混合舰种

- 航空战列舰、航空巡洋舰按主要水面身份选择现有 `type`，航空能力通过 `aircraft` 表达。
- 只有独立 UI 分类、资源路由或玩法规则确实需要时，才扩展 Go 枚举和加载器。
- 图片保留甲板、弹射器和舷外平台的原始比例；配置 `width` 仍使用舰体宽度。
- 不为单舰修改全局起降、返航或回收逻辑。`TestAll` 只按主 `type` 分组一次。

### 7. 更新图鉴与 TestAll

- 在所有正式语言的 references 文件中添加同名条目：简短历史或背景、作者以及资料链接；规格、游戏中实际武装和飞机配置由结构化字段承载。
- `description` 不得重复 `totalHP`、`length`、`width`、`maxSpeed`、`armaments`、`aircraft` 或“本作配置/装备”等字段；只保留历史身份、背景、特殊状态和必要的素材假设。四种语言保持同一信息层级，避免再次出现重复说明书式文案。
- 每个舰船图鉴的 `links` 统一只保留两项且顺序固定：第一项是用户提供或原作者发布的素材来源；第二项是目标语言维基百科中与该舰、舰级或型号最直接相关的条目。目标语言没有对应条目时，第二项回退到英文维基百科。
- 用 Wikipedia API、搜索结果或实际打开页面确认维基链接存在或可重定向；不得凭标题猜测 URL。四种语言允许因本地化维基条目不同而使用不同的第二项 URL，但第一项素材来源必须保持一致。
- 在 `TestAll` 对应 `providedShipNames` 添加舰名，并在同舰种网格中增加一艘己方 `HA` 初始舰，避免重叠。
- 除非用户明确要求，不修改其他任务。

### 8. 验证

- 确认原尺寸备份和正式俯视/侧视 PNG 均存在，方向正确，缩放保持长宽比；正式图舰艏、舰艉不得有会使可见舰体明显小于 `length*4` 的透明留白。
- 确认炮塔标记项目 JSON 已由用户确认，且 `ships.json5` 中的 `posPercent`、左右舷和武器数量与导出结果一致。
- 搜索舰名，确认 `ships.json5`、`references.json5`、`TestAll` 和 PNG 文件名一致且没有重复定义。
- 确认新舰插入到了正确的国家/舰种/舰级位置（国家 → 舰种 → 舰级代际逆序），没有落到文件末尾或同舰种块尾部，也没有打乱同小节既有条目的相对顺序；需要时用工具打印各「国家 × 舰种」小节的舰船列表核对。
- 搜索所有武器、弹药、发射器和飞机引用，确认上游配置存在。
- 用 `view_image` 检查正式 PNG 的完整轮廓和透明边缘；用 `git diff --check` 检查文本问题。
- 运行 `go build ./pkg/...`。最终说明资料来源、配置选择和未验证的视觉区域。
