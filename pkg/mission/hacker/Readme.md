# hacker

`hacker` package 实现任务内的作弊终端。它提供一个类似命令行的输入层，并通过 `cheat` 子包把命令映射为对 `MissionState` 的直接修改。

## 进入和更新顺序

终端由 `manager` 创建和驱动：

- `manager.New` 中调用 `hacker.NewTerminal()` 创建终端。
- `manager.updateMissionStatus` 中按下 `LeftCtrl + LeftShift + Backquote` 时进入 `MissionInTerminal`。
- 进入终端时会播放作弊音效，并强制关闭舰船编组模式。
- `manager.Update` 在 `MissionInTerminal` 状态下调用 `m.updateTerminal()`。
- `updateTerminal()` 直接调用 `Terminal.Update(m.state)`。
- 终端绘制由 `drawer.Draw(screen, state, terminal)` 读取 `Terminal` 的缓冲区和当前输入完成。

终端模式不会推进战斗模拟。它只处理输入、执行命令，并在需要时修改任务状态。

## Terminal 结构

`Terminal` 保存终端的显示和输入状态：

- `ReservedLines`：缓冲区最多保留的行数。
- `LineSpacing`：行间距。
- `FontSize`：字体大小。
- `Font`：当前字体。
- `Color`：当前字体颜色。
- `History`：已执行命令历史。
- `HistoryIndex`：当前历史浏览位置。
- `Buffer`：输出缓冲区。
- `Input`：当前输入内容。

`Line` 表示缓冲区中的一行：

- `LineTypeOutput`：普通输出。
- `LineTypeInput`：用户输入。

`Line.String()` 会给输入行自动加上 `>>> ` 前缀，输出行保持原文本。

## 初始化

`NewTerminal()` 创建终端并设置默认值：

- 根据当前屏幕布局计算 `ReservedLines`。
- 默认字体大小为 `24`。
- 默认行间距为 `6`。
- 默认输入前缀为 `>>> `。
- 默认字体为 `font.JetbrainsMono`。
- 默认颜色为 `colorx.Gold`。
- 默认缓冲区包含欢迎文本：提示输入 `help` 查看命令。

`ReservedLines` 的计算方式是用屏幕高度除以单行高度，再取约 4/5 作为终端可保留行数。

## 当前输入显示

`CurInputString()` 返回当前输入行：

- 固定以 `>>> ` 开头。
- 后面接 `Input.String()`。
- 光标使用 `_` 模拟闪烁。
- 闪烁逻辑按 `time.Now().Unix() & 1` 判断，每秒切换一次。

这个函数只生成显示文本，不修改输入状态。

## 输入处理

`Update(misState)` 是终端每帧输入处理入口。

执行顺序：

1. 调用 `inpututil.AppendJustPressedKeys(nil)` 获取本帧刚按下的按键。
2. 如果没有按键，直接返回。
3. 逐个处理刚按下的按键。

按键行为：

- Enter：提交当前输入。
- Backspace：删除当前输入最后一个字节。
- Delete：清空当前输入。
- ArrowUp：向更早的历史命令移动。
- ArrowDown：向更新的历史命令移动。
- 其他按键：根据是否按住 Shift，从按键映射表写入字符。

Enter 的详细流程：

1. 读取 `Input.String()` 得到命令。
2. 把命令作为 `LineTypeInput` 追加到 `Buffer`。
3. 如果命令非空，追加到 `History`。
4. 调用 `execCommand(misState, cmd)` 执行命令。
5. 如果 `Buffer` 超过 `ReservedLines`，只保留最后 `ReservedLines` 行。
6. 清空当前输入。
7. 重置 `HistoryIndex` 为 0。

注意：空命令会被写入缓冲区，但不会执行、不会进入历史，也不会触发清理输入的分支。

## 按键映射

`mapping.go` 提供两个映射表：

- `keyCharMap`：普通按键到字符。
- `keyWithShiftCharMap`：按住 Shift 时的按键到字符。

支持内容包括：

- 数字键。
- 英文字母。
- 常见标点。
- 空格。

终端没有使用系统文本输入 API，而是显式维护按键到字符的映射。这样实现简单、可控，也方便限制作弊终端支持的输入字符范围。

## 历史命令

`fillHistory()` 根据 `HistoryIndex` 把历史命令填入当前输入：

- `HistoryIndex <= 0` 或没有历史时，重置为 0 并保持输入为空。
- `HistoryIndex` 最大不会超过历史长度。
- 实际填入的是 `History[historyLength-HistoryIndex]`。

因此：

- 第一次按上方向键会填入最近一条命令。
- 继续按上方向键会向更早命令移动。
- 按下方向键会向更新命令移动。
- 回到 0 时输入被清空。

## 命令执行分发

`execCommand(misState, cmd)` 负责把字符串命令分发到具体行为。

匹配顺序：

1. 内置退出命令：`:q`、`:wq`、`exit`、`quit`。
2. 内置清屏命令：`clear`。
3. 帮助命令：`help`。
4. 调试帮助命令：`debug`。
5. 设置颜色：命令前缀 `:set color`。
6. 设置字体：命令前缀 `:set font`。
7. 普通秘籍表 `cheat.Cheats`。
8. 调试秘籍表 `cheat.DebugCheats`。
9. 如果都不匹配，输出 `command not found: <cmd>`。

退出命令会先把 `MissionStatus` 改回 `MissionRunning`，然后通过 `fallthrough` 继续执行 `clear`，因此退出终端时会清空缓冲区。

秘籍匹配时，普通秘籍先于调试秘籍。每个 Cheat 通过自己的 `Match(cmd)` 判断是否匹配；匹配成功后调用 `Exec(misState)`，并把返回文本写入输出缓冲区。

无论秘籍是否匹配成功，普通命令分发结束后都会追加一个空行，让输出更易读。

## 帮助和显示设置

`showHelpText()` 输出普通帮助：

- 先列出内置命令。
- 再遍历 `cheat.Cheats`，输出每个普通秘籍的命令格式和描述。

`showDebugText()` 输出调试帮助：

- 提示调试命令仅供开发者使用。
- 遍历 `cheat.DebugCheats`，输出每个调试秘籍的命令格式和描述。

`setColorByCmd(cmd)` 修改终端颜色：

- 去掉 `:set color` 前缀后，把剩余内容交给 `colorx.GetColorByName`。
- 找到颜色则更新 `Terminal.Color`。
- 找不到则向缓冲区输出错误信息。

`setFontByCmd(cmd)` 修改终端字体：

- 去掉 `:set font` 前缀。
- 转小写并裁剪空格。
- `regular` 使用 `font.JetbrainsMono`。
- `italic` 使用 `font.JetbrainsMonoItalic`。
- 其他值回退到 `font.JetbrainsMono`。

## Cheat 接口

`cheat.Cheat` 是所有秘籍的统一接口：

- `String()`：返回命令展示文本。
- `Desc()`：返回帮助说明。
- `Match(cmd)`：判断输入命令是否匹配。
- `Exec(state)`：执行效果并返回输出文本。

注册表：

- `Cheats`：普通秘籍，显示在 `help` 中。
- `DebugCheats`：调试秘籍，显示在 `debug` 中。

新增秘籍的常规步骤：

1. 新增一个结构体实现 `Cheat`。
2. 在 `Match` 中定义匹配规则。
3. 在 `Exec` 中修改 `MissionState` 并返回日志。
4. 把实例加入 `Cheats` 或 `DebugCheats`。

## 命令匹配工具

`isCommandEqual(s1, s2)` 用于固定命令匹配：

- 使用正则把所有空白字符移除。
- 使用 `strings.EqualFold` 忽略大小写比较。

因此 `show me the money`、`ShowMeTheMoney`、`SHOW   ME THE MONEY` 都可以匹配同一个固定命令。

`normalizeCommandToken(s)` 用于动态参数匹配：

- 只保留 ASCII 数字和英文字母。
- 大写字母会转成小写。
- 其他字符全部丢弃。

当前主要用于 `show me the ship <name>`，让舰船名称匹配不受大小写、空格和符号影响。

## 对象生成秘籍

`ShowMeTheDuck`

- 命令：`show me the duck`
- 在当前鼠标地图位置生成 `duck`。
- 如果当前位置是陆地，返回失败文本。
- 使用当前玩家的舰船 UID 生成器。
- 新舰船加入 `Arena.Ships`，所属阵营为当前玩家。

`ShowMeTheWaterdrop`

- 命令：`show me the waterdrop`
- 在当前鼠标地图位置生成 `waterdrop`。
- 不检查陆地。
- 使用当前玩家 UID 生成器，并加入 `Arena.Ships`。

`ShowMeTheMolaMola`

- 命令：`show me the molamola`
- 在当前鼠标地图位置生成 `molamola`。
- 如果当前位置是陆地，返回失败文本。
- 使用当前玩家 UID 生成器，并加入 `Arena.Ships`。

`ShowMeTheShip`

- 命令：`show me the ship <name>`
- 动态匹配 `<name>`，会遍历 `objUnit.AllShipNames`。
- 匹配时对命令和舰船名做 token 归一化。
- 位置在陆地上时拒绝创建。
- 创建成功后把命名舰船加入 `Arena.Ships`。
- `shipName` 暂存在 Cheat 实例字段中，随后由 `Exec` 使用。

`BlackGoldRush`

- 命令：`black gold rush`
- 在当前鼠标地图位置生成油井。
- 如果当前位置是陆地，返回失败文本。
- 油井半径为 `3 + rand.Intn(4)`，即 3 到 6。
- 油井产量为 `50 + rand.Intn(100)`，即 50 到 149。
- 新油井加入 `Arena.OilPlatforms`。

## 经济秘籍

`ShowMeTheMoney`

- 命令：`show me the money`
- 当前玩家资金增加 `10000`。
- 返回增加后的当前资金。

## 效果类秘籍

`AngelicaSinensis`

- 命令：`angelica sinensis`
- 当前只返回一段文本。
- 代码中保留了后续实现 TODO。

`BlackSheepWall`

- 命令：`black sheep wall`
- 设计目标是移除战争迷雾并显示所有敌军。
- 当前返回 `Not Implemented`。

`BathtubWar`

- 命令：`bathtub war`
- 把地图上所有战舰替换为 `duck`。
- 会先取出当前所有舰船，再重建 `Arena.Ships`。
- 新 duck 保留原舰船位置、旋转和所属阵营。
- UID 使用对应阵营的舰船 UID 生成器重新生成。

`WhoIsCallingTheFleet`

- 命令：`who is calling the fleet`
- 遍历当前玩家所属增援点。
- 把每个待入场舰船的 `TimeCost` 设置为 1。
- 只影响已经在 `OncomingShips` 队列中的舰船。

`DoNotDie`

- 命令：`do not die`
- 遍历当前选中的舰船。
- 如果舰船仍在 `Arena.Ships` 中，就把 `CurHP` 设置为 `Tonnage`。
- 这里依赖当前实现中“满血数值等于吨位”的约定。

`YouHaveBetrayedTheWorkingClass`

- 命令：`you have betrayed the working class`
- 遍历当前选中的舰船。
- 把存在的舰船阵营改为 `faction.ComputerAlpha`。

`AbandonDarkness`

- 命令：`abandon darkness`
- 遍历所有舰船。
- 只要舰船在当前相机视野内，就把所属阵营改为当前玩家。
- 不区分原本是否敌方，因此视野内己方舰船也会被重新赋值为当前玩家。

`Expelliarmus`

- 命令：`expelliarmus`
- 遍历所有舰船。
- 只处理当前相机视野内、且不属于当前玩家的舰船。
- 把目标舰船的 `Weapon` 设置为空 `objUnit.ShipWeapon{}`。

## 调试秘籍

`DebugAll`

- 命令：`debug all`
- 一次性开启所有当前定义的调试标志：
  - `DamageColorByTeam`
  - `ShowCursorPosObjInfo`
  - `ShowPlaneHP`

`DamageColorByTeam`

- 命令：`damage color by team`
- 切换 `misState.UI.DebugFlags.DamageColorByTeam`。
- 影响伤害数字按队伍着色的显示逻辑。

`ShowCursorPosObjInfo`

- 命令：`show cursor pos obj info`
- 切换 `misState.UI.DebugFlags.ShowCursorPosObjInfo`。
- 用于显示光标位置和悬停对象调试信息。

`ShowPlaneHP`

- 命令：`show plane hp`
- 切换 `misState.UI.DebugFlags.ShowPlaneHP`。
- 用于显示飞机当前 HP 和总 HP。

## 当前实现特点

- 终端输入和秘籍效果解耦，结构清晰，扩展成本低。
- 命令使用接口注册表驱动，不需要在终端里硬编码每个秘籍的效果。
- 固定命令支持忽略大小写和空白，对玩家输入比较宽容。
- 动态舰船生成命令使用 token 归一化，可以兼容配置名里的空格和符号差异。
- `Terminal` 直接持有显示状态，绘制层可以简单读取缓冲区和当前输入。
- 秘籍 `Exec` 直接修改 `MissionState`，实现直接有效，但也意味着后续如果要支持撤销、审计或多人同步，需要额外抽象命令效果。
- `ShowMeTheShip` 会把匹配到的 `shipName` 存在 Cheat 实例字段中。当前终端单线程调用没有问题；如果未来并发执行命令，需要避免这种可变共享状态。
