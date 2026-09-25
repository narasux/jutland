# Turret Position Marker

这是一个本地浏览器工具，用于在战舰**俯视图**上点击标记火炮的纵向位置，并生成 `posPercent`。

## 依赖

- Python 3
- Pillow
- 现代浏览器

当前实现使用 Python 标准库启动本地 HTTP 服务，不依赖 `tkinter` 或 PyQt。

## 启动

在仓库根目录执行：

```bash
python3 utils/turret_marker/server.py resources/images/ships/top/battleship/bretagne.png
```

也可以通过 Makefile：

```bash
make turret-marker IMAGE=resources/images/ships/top/battleship/bretagne.png
```

浏览器会自动打开类似 `http://127.0.0.1:8765/` 的地址。若不想自动打开浏览器：

```bash
python3 utils/turret_marker/server.py path/to/top.png --no-open
```

如果纵向俯视图过长，可以在浏览器中顺时针横置预览，源图片不会被改写：

```bash
make turret-marker IMAGE=path/to/top.png ROTATE=90
```

推荐在启动时同时给出武器类型和预期数量，界面会显示“已标/应有”，并阻止靠肉眼漏项：

```bash
make turret-marker \
  IMAGE=path/to/top.png \
  ROTATE=90 \
  TYPES='main:5,secondary:14,aa75:8,aa132:4,aa20:7,aa20d:2'
```

只检查图片边界，不启动服务：

```bash
python3 utils/turret_marker/server.py path/to/top.png --check
```

## 使用流程

1. 使用**俯视图**，不要使用侧视图或上下两张合图。默认舰艏在图片上方。
2. 图片会按原始比例放大到画布最大宽度，舰体长轴随之拉到最长；超出画布的部分可滚动查看。
3. 工具会根据 Alpha 边界自动识别舰体首尾端点；如识别不准，点击“手动校准”，分别点击两个端点，顺序不限。
4. 选择武器类型，然后点击炮塔位置。
5. 右侧列表显示每个标记的 `posPercent`，可删除错误标记。
6. 点击“复制当前类型”只复制当前武器类型的 `posPercent`，每行一个数值；点击“复制全部”复制所有标记的数值。
7. 点击“复制项目 JSON”导出包含类型、左右舷、像素坐标、两位小数 `posPercent`、射界和数量校验的结构化结果。

`posPercent` 的计算与运行时一致：以校准中心为 `0`，向舰艏为正、向舰艉为负，范围约为 `-1..1`。默认舰艏在上方；如果舰艏在下方，启动时增加 `--bow bottom`。

标记状态会自动保存在当前浏览器的本地存储中，刷新页面不会丢失。若图片的长轴端点贴边，页面顶部状态栏会显示裁切警告。

## 对称火炮

“对称吸附”默认开启。连续点击同一类型的两个对称炮塔时，如果两次换算结果的差值小于吸附阈值，第二次会复用第一次的精确 `posPercent`，避免手工点击误差造成同组炮塔数值不一致。

需要独立记录两个相近位置时，可以取消勾选，或调低吸附阈值。

## 射界参考

右侧面板会根据武器类型和当前位置给出常见射界推荐，例如主炮前向、后向和中段射界，也可以人工修正。该面板只用于参考和会话内校正，**不会进入“复制 posPercent”输出**。

推荐值参考了 `configs/ships.json5` 中同类舰船的常见写法：

- 前向主炮：`right [0, 150]`、`left [210, 360]`
- 后向主炮：`right [30, 180]`、`left [180, 330]`
- 中段/两舷武器：`right [0, 180]`、`left [180, 360]`
- 鱼雷：`right [30, 150]`、`left [210, 330]`

具体数值仍应以同型舰和实际炮位为准。
炮廓炮和其他舷侧武器会按标记的左右舷只保留一侧射界；另一侧以 `[0, 0]` 或 `[360, 360]` 禁用。

## 命令行参数

```text
--host HOST                 默认 127.0.0.1
--port PORT                 默认 8765
--bow top|bottom|left|right 舰艏方向，默认 top
--rotate 0|90|180|270       浏览器预览顺时针旋转角度，默认 0
--types TYPE[:COUNT],...     武器类型及预期数量，默认 main,secondary,aa,torpedo,rocket,custom
--check                     只打印图片边界并退出
--no-open                   不自动打开浏览器
```
