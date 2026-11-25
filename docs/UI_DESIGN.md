# MysteryVerse UI/UX 设计规范

> 文档状态：Design Ready
> 适用阶段：React 前端实现、视觉评审、Playwright 验收
> 设计范围：完整产品体验与实现契约，不包含业务代码
> 目标技术栈：React + TypeScript + Vite + Tailwind CSS + Zustand + TanStack Query + Motion
> 主要依据：[`MysteryVerse-PRD.md`](./MysteryVerse-PRD.md)、[`TECH_DESIGN.md`](./TECH_DESIGN.md)、[`DATABASE_DESIGN.md`](./DATABASE_DESIGN.md)、[`API_DESIGN.md`](./API_DESIGN.md)

> **2026-11-21 架构基线补充**：ADR-003、ADR-009、ADR-010 为 Accepted；
> ADR-002、ADR-004、ADR-006、ADR-007、ADR-008 为 Proposed。
> Proposed 协议和页面契约在负责人确认前不得作为实现冻结依据。

## 0. 设计结论

### 0.1 体验主张

MysteryVerse 不是一个套着悬疑皮肤的 Web App，而是一个「正在被操作的故事现场」。界面需要同时让用户感到：

1. 自己在处理一份有来源、有权限、有因果的档案。
2. 自己正跨过一道门，进入某个房间、身份、阶段或世界。
3. 信息之间存在可追踪的线索关系，而不是互不相关的卡片。
4. 私密信息被明确封印，只有服务端授权的状态才能开启。

共享视觉隐喻：

| 系统 | 表达 | 使用位置 |
| --- | --- | --- |
| Archive / 档案 | 纸张、卷宗、编号、来源、时间戳、证据夹 | 剧本、线索、复盘、创作者工具 |
| Threshold / 门槛 | 门框、阶段切换、房间入口、场景边界 | 首页模式入口、房间、剧情转场 |
| Thread / 线索 | 有向关系、证据连线、时间线、因果提示 | 人物关系、证据板、推理对照 |
| Seal / 封印 | 蜡封、金属锁标、可见范围、揭示门控 | 私密角色、身份、真相、投票确认 |

品牌材料应服务于信息层级，不作为装饰堆叠：

- 纸张承载人的证词、剧本和线索。
- 金属承载系统规则、阶段、计时和不可逆确认。
- 光影承载地点、时间与戏剧状态。
- 线框与编号承载关系、来源和可回放因果。

### 0.2 一个页面只回答一个当前问题

| 页面 | 当前问题 |
| --- | --- |
| 首页 | 我现在能玩什么，如何最快进入下一步？ |
| 大厅 | 哪些内容适合我的人数、时长和偏好？ |
| 剧本详情 | 这个剧本是否适合本局，能否创建房间？ |
| 工坊 | 我要用什么素材生成哪种可玩内容？ |
| 生成进度 | 系统做到哪一步，出了什么问题，我下一步能做什么？ |
| 编辑器 | 哪些对象有问题，如何修复并安全发布？ |
| 房间等待 | 谁已就位，还差什么条件才能开局？ |
| 剧本杀 | 当前阶段是什么，我知道了什么，合法行动是什么？ |
| 狼人杀 | 昼夜与轮次是什么，我的私密信息是什么，当前能做什么？ |
| RPG | 我在哪里，世界发生了什么，我可以如何行动？ |
| 结算 | 规则确认的结果是什么，为什么？ |
| 复盘 | 关键因果链是什么，我错过了什么？ |

### 0.3 设计优先级

设计冲突时按以下顺序决策：

1. 服务端权威状态与隐私边界。
2. 当前阶段、计时和主操作的可读性。
3. 桌面与移动端关键流程可完成性。
4. 剧情沉浸与氛围表达。
5. 非关键装饰、声音和持续环境动效。

### 0.4 非目标与设计红线

禁止：

- 普通 SaaS Dashboard 模板、等权统计卡片和通用侧边栏。
- 全站紫色渐变、蓝色霓虹或玻璃拟态。
- 用大量圆角卡片分割所有页面区域。
- 卡片内再套卡片、装饰性容器层层包裹。
- 用 Emoji 替代 Lucide 等专业图标系统。
- 在未确认服务端结果前展示线索获得、投票成功、淘汰或胜负。
- 用模糊、占位剪影或动效提示未授权私密内容。
- 由动画回调、客户端计时器或 AI 文本决定游戏阶段和结果。
- 用持续粒子、循环扫光或视差效果覆盖阅读和决策区域。
- 用负字距或视口宽度驱动的字号缩放。

### 0.5 设计系统分层

1. Primitive Tokens：原始颜色、字号、间距、圆角、时长。
2. Semantic Tokens：背景、文本、证据、公开、私密、危险、焦点。
3. Component Contracts：按钮、输入、卡、抽屉、阶段条、证据组件。
4. Page Patterns：首页、游戏壳、编辑工作台、结算与复盘。

业务组件只消费 Semantic Tokens；Primitive Tokens 只存在于主题层。

---

## 1. 视觉定位

### 1.1 共享品牌语言

产品整体采用「暗色操作框架 + 暖色证据材料 + 冷色环境光」的混合温度系统。默认游戏体验使用 **Archive Night**，创作与长阅读使用 **Paper Daylight**，具体地点可使用局部 **Scene Override**。

视觉层级：

- Operational Layer：阶段、计时、主操作、连接状态、隐私范围。
- Interpretive Layer：叙述、角色、线索、关系、投票、结果。
- Atmospheric Layer：场景、光影、雨雾、低密度粒子、声音。

Operational Layer 在视觉上必须始终可读、可聚焦、可到达。Atmospheric Layer 不得遮住当前动作，也不得成为唯一状态提示。

### 1.2 三模式统一与差异化

| 模式 | 主构图 | 核心材料 | 主焦点 | 禁止 |
| --- | --- | --- | --- | --- |
| 剧本杀 | 调查档案工作台 | 纸张、墨色、琥珀证据、金属阶段标 | 当前场景与证据关系 | 把所有线索做成相同圆角卡 |
| 狼人杀 | 仪式席位环 | 月银、雾、昼夜光、克制的牛血红 | 玩家席位与当前发言/投票 | 用边框颜色泄露隐藏身份 |
| RPG | 环境世界 + 三区导航 | 苔绿、铜、矿物、天气和地形 | 场景主体与当前行动 | 背景图后接普通聊天应用 |

模式差异必须来自构图、信息模型和材料，不只来自标题和强调色。任意截图即使去掉模式标题，也应能辨认当前模式。

### 1.3 隐私视觉语言

| Scope | 形态 | 持久标签 | 补充信号 |
| --- | --- | --- | --- |
| Public | 开放表面、共享线、正常对比 | 公开 | 共享频道图标 + 文本 |
| Role-private | 内嵌封印面、暖锁标 | 仅你可见 | 锁图标 + 角色上下文 |
| Faction-private | 独立信道边界 | 阵营可见 | 阵营标签 + 独立频道 |
| Discoverer-private | 证据夹内页 | 发现者可见 | 持有人标记 + 来源 |
| System | 金属/工具样式 | 系统 | 系统图标、固定模板 |
| Unrevealed | 不展示或明确 sealed | 尚未揭示 | 不显示标题长度、缩略图或元数据 |

颜色不能单独承担可见范围。私密文本不得出现在共享截图背景、页面转场克隆层、Toast、加载骨架或通知预览中。

### 1.4 首页不是营销页

首页首屏以产品内容为 Hero：

- 有进行中房间时，继续游玩条优先。
- 三个模式入口使用三种不同构图的「门槛面板」，不是三张同构营销卡。
- 推荐剧本和热门房间应露出下一屏内容提示。
- 不设置占据首屏的大幅品牌广告或抽象氛围图。
- 未开放模式使用明确阶段文案与禁用状态，不制造可点击假入口。

### 1.5 Anti-AI-Slop 审计

每个主要视觉决策必须回答：

1. 它表达了哪一种产品状态或用户需求？
2. 删除后是否损失信息、方向或操作？
3. 它是否来自 MysteryVerse 的档案/仪式/世界逻辑？
4. 无动效、声音缺失、背景图失败时是否仍可理解？

交付前执行「移除一项」原则：从最强视觉区域删除一个非必要装饰；若界面更清楚，则保留删除结果。

---

## 2. 前端架构与状态边界

### 2.1 技术约束落地

| 技术 | 职责 | 禁止 |
| --- | --- | --- |
| React + TypeScript | 页面、组件、交互 | 业务规则判定 |
| Vite | 构建、按路由拆包 | 将私密数据写入构建期资源 |
| Tailwind CSS | Token 消费、布局、响应式 | 大量不可复用内联样式 |
| Zustand | UI 状态、实时投影副本、草稿输入 | 权威身份、权限、胜负 |
| TanStack Query | REST 数据、缓存、分页、mutation | 代替 WebSocket 权威事件流 |
| Motion | 组件进出、布局、剧情编排 | 决定游戏结果或阶段 |
| WebSocket | 房间与游戏实时事件 | 承载生成任务主进度 |
| SSE | LangGraph 生成进度 | 替代游戏权威事件 |
| React Router | 路由与体验守卫 | 作为最终权限校验 |

### 2.2 状态所有权

| 状态 | 归属 | 示例 | 生命周期 |
| --- | --- | --- | --- |
| REST 服务端状态 | TanStack Query | 剧本详情、个人资料、历史、社区列表 | Query 缓存 |
| 游戏首屏投影 | TanStack Query → Zustand 初始化 | `GET /games/{id}/projection` | 进入房间后 |
| 实时游戏投影 | Zustand + reducer | 阶段、公开事件、本人私密状态 | 内存，退出/结束后清理 |
| 纯 UI 状态 | Zustand | Tab、抽屉、选中证据、表单草稿 | 页面/会话 |
| 路由状态 | URL / React Router | 筛选、房间 ID、页面路径 | 可分享、可刷新 |
| 表单临时状态 | React local / 小型 Store | 编辑器字段、上传进度 | 页面 |
| 权威状态 | Go 后端 | 身份、线索 ACL、投票、胜负、揭示门控 | 服务端 |

规则：

- 游戏首屏必须先获取 REST 快照，再写入 `gameStore`。
- WebSocket 事件只通过 reducer 修改投影。
- 重连使用服务端快照整体替换投影，不做本地双向合并。
- D2-D4 投影只保存在内存，不写入 LocalStorage、IndexedDB 或 URL。
- 关键游戏操作不做乐观成功；UI 只显示 `submitting`，等待服务端确认。

### 2.3 建议 Zustand Slices

```ts
type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "syncing"
  | "ready"
  | "degraded"
  | "reconnecting"
  | "offline";

type GameStore = {
  connectionState: ConnectionState;
  lastRoomSeq: number;
  lastRoomChatSeq: number;
  lastGameSeq: number;
  stateVersion: number;
  viewer: { playerId: string; roleId?: string };
  phase: { key: string; deadlineAt?: string };
  publicState: unknown;
  privateState: unknown;
  availableActions: string[];
  ui: {
    activeMobileView: "story" | "clues" | "chat" | "role";
    selectedClueId?: string;
    selectedEvidenceIds: string[];
    drawer?: string;
    unreadByChannel: Record<string, number>;
  };
};
```

页面级 Store 分离：

- `gameStore`：当前游戏投影与 UI 焦点。
- `roomStore`：等待房间席位、准备、邀请 UI 状态。
- `workshopStore`：生成任务选择、预览 Tab、编辑器本地草稿。
- `lobbyStore`：仅保存抽屉/视图偏好；筛选真值放 URL。

### 2.4 TanStack Query Key 约定

```ts
const queryKeys = {
  me: (scope: string) => ["me", scope] as const,
  scripts: (filters: string, viewerScope: string) =>
    ["scripts", filters, viewerScope] as const,
  script: (scriptId: string, viewerScope: string) =>
    ["script", scriptId, viewerScope] as const,
  generationJob: (jobId: string) => ["generation-job", jobId] as const,
  editor: (versionId: string, objectSet: string) =>
    ["editor", versionId, objectSet] as const,
  room: (roomId: string, view: string) => ["room", roomId, view] as const,
  gameProjection: (gameId: string, viewerId: string) =>
    ["game-projection", gameId, viewerId] as const,
  gameResult: (gameId: string, view: string) =>
    ["game-result", gameId, view] as const,
  replay: (gameId: string, viewerRoleId: string, phaseKey: string) =>
    ["replay", gameId, viewerRoleId, phaseKey] as const,
};
```

Query Key 必须包含权限上下文相关 ID。切换角色视角、房间席位或 viewer scope 时不得复用旧缓存。

### 2.5 WebSocket 与 SSE 分工

```text
REST projection
  → Zustand gameProjection
  → WebSocket room/game seq
  → reducer
  → UI
```

- `room`、`room_chat` 与 `game` 流使用独立 `seq`，不可交叉比较。
- 客户端保存 `lastRoomSeq`、`lastRoomChatSeq`、`lastGameSeq`、`last_state_version`。
- 序号缺口立即发送 `resync.request`，通过 HTTP 快照恢复。
- AI DM/NPC 文本使用 `ai_stream`，不推进游戏 `seq` 或 `state_version`。
- 生成进度使用 `GET /generation-jobs/{job_id}/events` SSE，按 `last_event_id` 重连。
- 规则结果、阶段、投票、淘汰、资源变化必须先于正式消息流更新。

### 2.6 动画引擎选择

| 需求 | 引擎 | 说明 |
| --- | --- | --- |
| Hover、Press、Focus、简单展开 | CSS transition | 默认首选 |
| React Presence、布局变化、抽屉、Tab 指示、证据列表连续性 | Motion | 统一动画入口 |
| 长序列剧情、多个事件链精确拍点 | Motion variants / 状态机 | 当前阶段不引入 GSAP |
| 高密度关系图/地图交互 | SVG 优先 | 节点多到影响 DOM 时再评估 Canvas |
| 3D 场景 | 非 MVP 必需 | 不因单个效果引入 Three.js |

全应用使用 `MotionConfig reducedMotion="user"`。复杂序列必须有跳过、中断和静态终帧。

### 2.7 权限与数据安全

- 前端只渲染服务端已投影 DTO。
- 图标、按钮隐藏只改善体验，不构成权限。
- 私密截图风险区不得包含他人的头像、私聊预览或未授权线索标题。
- 分享战绩必须使用服务端脱敏分享卡片。
- 文件上传使用短期 presigned URL；上传进度与扫描状态分开显示。
- AI 进度只展示脱敏节点、错误和产物摘要，不展示 Prompt、原文块或内部推理。

---

## 3. Design System

### 3.1 色彩系统

#### Primitive Palette

| Token | 色值 | 用途 |
| --- | --- | --- |
| `ink-950` | `#0B0D10` | 游戏画布 |
| `ink-900` | `#14181D` | 主面板 |
| `ink-850` | `#1D232A` | 选中区域、抬升面 |
| `ink-800` | `#090B0E` | 内嵌凹陷面 |
| `bone-100` | `#E8E3D8` | 暗色主题主文本 |
| `bone-300` | `#C8C1B4` | 次要长文本 |
| `ash-500` | `#A7AAA8` | 辅助文本 |
| `metal-600` | `#596166` | 强边框、图标 |
| `metal-700` | `#2C333A` | 细边框 |
| `paper-50` | `#F1EDE4` | 明亮画布 |
| `paper-100` | `#E6DFD2` | 卷宗面 |
| `paper-200` | `#D8CEBD` | 纸张边界 |
| `amber-500` | `#C7783F` | 证据、发现、可操作线索 |
| `amber-300` | `#E2B36D` | Focus、高亮证据 |
| `green-500` | `#4D8273` | 已确认连接、合法可推导 |
| `red-500` | `#B34D4B` | 危险、冲突、不可逆 |
| `silver-400` | `#B7C3CC` | 狼人杀公共阶段 |
| `oxblood-600` | `#713B43` | 狼人私密阵营，克制使用 |
| `moss-500` | `#63775B` | RPG 环境 |
| `copper-500` | `#A46F4F` | RPG 器物、路线 |
| `blue-500` | `#6E93A6` | 系统信息、中性提示 |

同一视口只允许一个主强调色与一个状态强调色。模式色用于局部情境，不得覆盖公共/私密/危险等语义色。

#### Semantic Tokens：Archive Night

```css
:root,
[data-theme="archive"] {
  --mv-canvas: 11 13 16;
  --mv-surface: 20 24 29;
  --mv-surface-raised: 29 35 42;
  --mv-surface-inset: 9 11 14;
  --mv-surface-paper: 230 223 210;
  --mv-text-primary: 232 227 216;
  --mv-text-secondary: 167 170 168;
  --mv-text-muted: 128 134 135;
  --mv-text-disabled: 105 109 110;
  --mv-text-on-paper: 23 26 29;
  --mv-border-subtle: 44 51 58;
  --mv-border-strong: 89 97 102;
  --mv-action-primary: 212 162 92;
  --mv-action-primary-hover: 230 185 115;
  --mv-action-ink: 23 26 29;
  --mv-focus: 226 179 109;
  --mv-evidence: 199 120 63;
  --mv-state-success: 77 130 115;
  --mv-state-warning: 201 151 67;
  --mv-state-danger: 179 77 75;
  --mv-state-info: 110 147 166;
  --mv-public: 140 154 157;
  --mv-private: 212 162 92;
  --mv-secret: 113 59 67;
  --mv-system: 183 195 204;
  --mv-scene-glow: 226 179 109;
}
```

#### Semantic Tokens：Paper Daylight

```css
[data-theme="paper"] {
  --mv-canvas: 241 237 228;
  --mv-surface: 230 223 210;
  --mv-surface-raised: 248 245 238;
  --mv-surface-inset: 216 206 189;
  --mv-surface-paper: 230 223 210;
  --mv-text-primary: 23 26 29;
  --mv-text-secondary: 83 88 89;
  --mv-text-muted: 110 116 116;
  --mv-text-disabled: 142 145 142;
  --mv-text-on-paper: 23 26 29;
  --mv-border-subtle: 203 193 177;
  --mv-border-strong: 126 128 121;
  --mv-action-primary: 151 82 43;
  --mv-action-primary-hover: 176 99 54;
  --mv-action-ink: 248 245 238;
  --mv-focus: 151 82 43;
  --mv-evidence: 164 92 49;
  --mv-state-success: 54 106 91;
  --mv-state-warning: 148 103 25;
  --mv-state-danger: 153 54 52;
  --mv-state-info: 56 102 126;
  --mv-public: 91 105 108;
  --mv-private: 151 82 43;
  --mv-secret: 103 55 62;
  --mv-system: 91 100 104;
}
```

场景局部色只能覆盖：

- 环境光、背景、前景天气、场景材质。
- 不覆盖 Focus、Danger、Success、Public/Private 标签。

### 3.2 字体系统

| 角色 | 字体栈 | 使用 |
| --- | --- | --- |
| Narrative / Display | `"Noto Serif SC", "Source Han Serif SC", "Songti SC", Georgia, serif` | 场景标题、案件标题、关键台词 |
| UI / Body | `"Noto Sans SC", "Source Han Sans SC", "PingFang SC", Inter, system-ui, sans-serif` | 导航、按钮、表单、正文 |
| Data / Mono | `"IBM Plex Mono", "SFMono-Regular", Consolas, monospace` | 计时、编号、事件序号、证据时间 |

字体规则：

- 自托管字体使用 WOFF2、`font-display: swap`，中文字体按实际字形子集化。
- 中文正文行高不低于 1.6；密集 UI 不低于 1.4。
- `letter-spacing: 0` 用于常规 UI。
- 仅短标签可使用正字距，最大 `0.08em`。
- 数字计时、票数、序号使用 `font-variant-numeric: tabular-nums`。

### 3.3 字号与行高

| Token | Desktop | Compact | 用途 |
| --- | --- | --- | --- |
| `text-display` | 56/64 | 38/46 | 真相、结局、结果标题 |
| `text-h1` | 40/48 | 32/40 | 页面主标题 |
| `text-h2` | 28/36 | 24/32 | 区块标题 |
| `text-h3` | 22/30 | 20/28 | 面板标题 |
| `text-body-lg` | 18/32 | 17/30 | 剧情阅读 |
| `text-body` | 16/28 | 16/28 | 默认正文 |
| `text-ui` | 14/20 | 14/20 | 控件与密集 UI |
| `text-caption` | 12/18 | 12/18 | 元数据、标签 |
| `text-micro` | 11/16 | 11/16 | 非关键辅助信息 |

正文阅读宽度 45-72 个中文字符；侧栏 28-42 个字符。移动端不压缩桌面字号，不采用 `vw` 字号。

### 3.4 间距与网格

基础单位 4px：

```text
0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128
```

| Token | 值 |
| --- | --- |
| `space-1` | 4px |
| `space-2` | 8px |
| `space-3` | 12px |
| `space-4` | 16px |
| `space-5` | 20px |
| `space-6` | 24px |
| `space-8` | 32px |
| `space-10` | 40px |
| `space-12` | 48px |
| `space-16` | 64px |
| `space-20` | 80px |
| `space-24` | 96px |

桌面页面：

- 12 列网格，24px gutter。
- 普通内容最大宽度 1600px。
- 阅读列最大 72ch。
- 编辑器与游戏壳可使用全宽工作区，但操作内容受 min/max 宽度约束。

移动端：

- 4 列网格，16px side padding。
- 底部操作区考虑 `env(safe-area-inset-bottom)`。
- 二级面板转换为 Tab、Sheet 或 Drawer。

### 3.5 圆角、边框、阴影

```css
:root {
  --mv-radius-xs: 2px;
  --mv-radius-control: 6px;
  --mv-radius-card: 8px;
  --mv-radius-overlay: 12px;
  --mv-border-hairline: 1px;
  --mv-border-emphasis: 2px;
  --mv-shadow-card: 0 1px 0 rgb(255 255 255 / 0.04),
                   0 8px 24px rgb(0 0 0 / 0.18);
  --mv-shadow-overlay: 0 20px 64px rgb(0 0 0 / 0.42);
  --mv-shadow-inset: inset 0 1px 0 rgb(255 255 255 / 0.04);
}
```

规则：

- 卡片圆角不超过 8px，Overlay 可用 12px。
- 一个普通面板只使用边框、阴影、材质或色阶中的一种主要分隔方式。
- 卷宗可用切角、折痕、装订孔、印章提示材料，但不强行做成卡片。
- Focus 环不能被 `overflow: hidden` 截断。

### 3.6 层级与密度

| Token | z-index | 用途 |
| --- | --- | --- |
| `z-base` | 0 | 场景与正文 |
| `z-sticky` | 20 | 阶段条、保存条 |
| `z-overlay` | 40 | 抽屉、Popover |
| `z-modal` | 60 | Dialog、证据揭示 |
| `z-toast` | 80 | 非关键反馈 |
| `z-debug` | 100 | 仅开发环境 |

密度：

- Browse：舒适，列表行高 64-88px。
- Editor/Admin：紧凑，控件高度 32-40px。
- Gameplay：中密度，主操作高度 44-52px。
- Narrative：低密度，长文本间距优先。

### 3.7 图标

- 统一使用 Lucide；项目已使用其他 React 图标系统时只保留一套。
- 尺寸：16px inline、18-20px 控件、24px 导航/强调。
- 图标不代替标签；陌生图标需 Tooltip + accessible name。
- 角色、线索、证据不得使用 Emoji。
- 私密/公开标记必须图标 + 文本，不只变色。

### 3.8 动效 Tokens

```css
:root {
  --mv-duration-quiet: 140ms;
  --mv-duration-fast: 180ms;
  --mv-duration-structural: 280ms;
  --mv-duration-panel: 360ms;
  --mv-duration-reveal: 700ms;
  --mv-duration-cinematic: 1800ms;
  --mv-ease-standard: cubic-bezier(0.2, 0, 0, 1);
  --mv-ease-enter: cubic-bezier(0.16, 1, 0.3, 1);
  --mv-ease-exit: cubic-bezier(0.4, 0, 1, 1);
  --mv-ease-emphasis: cubic-bezier(0.34, 1.2, 0.64, 1);
  --mv-ease-linear: linear;
}
```

| Tier | 时长 | 用途 |
| --- | --- | --- |
| Quiet UI | 80-220ms | Hover、Press、Focus、Tooltip |
| Structural | 180-420ms | 抽屉、Dialog、Tab、页面内容 |
| Narrative | 500-1400ms | 线索、角色、关系、证据解锁 |
| Cinematic | 1200-3000ms | 昼夜、地点、真相、隐藏结局 |

### 3.9 Tailwind 配置设计

以下以 Tailwind CSS 配置式主题为例；若采用 Tailwind v4，可将相同变量映射到 `@theme`。

```ts
// tailwind.config.ts
import type { Config } from "tailwindcss";

const rgb = (name: string) => `rgb(var(${name}) / <alpha-value>)`;

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: rgb("--mv-canvas"),
        surface: {
          DEFAULT: rgb("--mv-surface"),
          raised: rgb("--mv-surface-raised"),
          inset: rgb("--mv-surface-inset"),
          paper: rgb("--mv-surface-paper"),
        },
        content: {
          primary: rgb("--mv-text-primary"),
          secondary: rgb("--mv-text-secondary"),
          muted: rgb("--mv-text-muted"),
          disabled: rgb("--mv-text-disabled"),
          paper: rgb("--mv-text-on-paper"),
        },
        border: {
          subtle: rgb("--mv-border-subtle"),
          strong: rgb("--mv-border-strong"),
        },
        action: {
          DEFAULT: rgb("--mv-action-primary"),
          hover: rgb("--mv-action-primary-hover"),
          ink: rgb("--mv-action-ink"),
        },
        focus: rgb("--mv-focus"),
        evidence: rgb("--mv-evidence"),
        state: {
          success: rgb("--mv-state-success"),
          warning: rgb("--mv-state-warning"),
          danger: rgb("--mv-state-danger"),
          info: rgb("--mv-state-info"),
        },
        scope: {
          public: rgb("--mv-public"),
          private: rgb("--mv-private"),
          secret: rgb("--mv-secret"),
          system: rgb("--mv-system"),
        },
      },
      fontFamily: {
        narrative: [
          "Noto Serif SC",
          "Source Han Serif SC",
          "Songti SC",
          "Georgia",
          "serif",
        ],
        ui: [
          "Noto Sans SC",
          "Source Han Sans SC",
          "PingFang SC",
          "Inter",
          "system-ui",
          "sans-serif",
        ],
        mono: ["IBM Plex Mono", "SFMono-Regular", "Consolas", "monospace"],
      },
      fontSize: {
        display: ["3.5rem", { lineHeight: "4rem" }],
        h1: ["2.5rem", { lineHeight: "3rem" }],
        h2: ["1.75rem", { lineHeight: "2.25rem" }],
        h3: ["1.375rem", { lineHeight: "1.875rem" }],
        "body-lg": ["1.125rem", { lineHeight: "2rem" }],
        body: ["1rem", { lineHeight: "1.75rem" }],
        ui: ["0.875rem", { lineHeight: "1.25rem" }],
        caption: ["0.75rem", { lineHeight: "1.125rem" }],
        micro: ["0.6875rem", { lineHeight: "1rem" }],
      },
      spacing: {
        1: "4px",
        2: "8px",
        3: "12px",
        4: "16px",
        5: "20px",
        6: "24px",
        8: "32px",
        10: "40px",
        12: "48px",
        16: "64px",
        20: "80px",
        24: "96px",
      },
      borderRadius: {
        xs: "2px",
        control: "6px",
        card: "8px",
        overlay: "12px",
      },
      boxShadow: {
        card: "0 1px 0 rgb(255 255 255 / 0.04), 0 8px 24px rgb(0 0 0 / 0.18)",
        overlay: "0 20px 64px rgb(0 0 0 / 0.42)",
      },
      screens: {
        compact: { max: "639px" },
        medium: { min: "640px", max: "1023px" },
        wide: { min: "1024px", max: "1439px" },
        ultrawide: { min: "1440px" },
      },
      transitionTimingFunction: {
        standard: "cubic-bezier(0.2, 0, 0, 1)",
        enter: "cubic-bezier(0.16, 1, 0.3, 1)",
        exit: "cubic-bezier(0.4, 0, 1, 1)",
        emphasis: "cubic-bezier(0.34, 1.2, 0.64, 1)",
      },
    },
  },
} satisfies Config;
```

使用示例：

```tsx
<button className="h-11 rounded-control bg-action px-4 font-ui text-ui font-semibold text-action-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus">
  提交搜证
</button>
```

禁止在业务组件内散落 `#hex`、任意 `text-[13.7px]`、重复阴影或无 Token 渐变。

---

## 4. 组件设计

### 4.1 基础组件契约

#### Button

| 项 | 规范 |
| --- | --- |
| Variant | `primary`、`secondary`、`quiet`、`destructive` |
| Size | `sm: 32px`、`md: 44px`、`lg: 52px` |
| 状态 | Default、Hover、Focus-visible、Pressed、Loading、Success、Error、Disabled |
| Loading | 保持原宽度和高度，文字或图标局部替换，禁止重复提交 |
| Disabled | 必须可在附近或 Tooltip 中看到原因 |
| 图标 | Lucide；仅熟悉操作可 Icon-only，必须有 accessible name |

#### Text Input / Textarea

- 可见 Label，不以 Placeholder 代替。
- Error 与帮助文本和字段建立 `aria-describedby`。
- 失败后保留输入。
- 自由行动输入显示“理解中、需澄清、不可执行、已结算、叙述中”中至少一个明确状态。
- 聊天输入与游戏主操作区域保持稳定高度，不能因 AI 流式文本重排而跳动。

#### Tabs / Segmented Control

- 用于同一对象上下文内的互斥视图。
- 移动端最多直接并排 4 个短标签；更多选项使用菜单或可滚动 Tab Rail。
- Indicator 动画 180-260ms，不延迟内容可读。
- 当前 Tab、焦点和对应 Panel 有正确 ARIA 关系。

#### Dialog / Sheet / Drawer

- 必须有标题、焦点陷阱、Escape 关闭和焦点返回。
- 不可逆操作明确对象、后果和是否可撤销。
- 移动 Sheet 保留场景上下文，不遮挡固定主操作。
- 电影式揭示仍需要可关闭/跳过路径。

#### Status / Timer / Progress

- 计时使用服务端 `deadlineAt` 与服务器时间偏移，不使用客户端动画判断阶段结束。
- 计时使用 tabular numerals。
- 剩余时间进入预警区间时同时使用文本、形状和颜色。
- 多阶段任务显示当前节点、完成节点、失败节点、重试次数和最终结果。

#### Privacy Badge

```ts
type VisibilityScope =
  | "public"
  | "room"
  | "role"
  | "role_group"
  | "player"
  | "runtime"
  | "audit";
```

显示规则：

- `public` → 公开
- `room` → 本房间可见
- `role` → 仅角色可见
- `role_group` → 阵营可见
- `player` → 仅你可见 / 发现者可见
- `runtime` / `audit` → 不在普通玩家 UI 呈现

### 4.2 GameRoom

**职责**

为三种游戏提供稳定外壳，承载阶段、隐私范围、主区、侧区、快捷入口和连接状态。GameRoom 不决定布局内容，只负责区域、层级和状态保护。

**Props**

```ts
type GameRoomProps = {
  roomId: string;
  gameId: string;
  mode: "script_murder" | "werewolf" | "rpg";
  phase: { key: string; title: string; deadlineAt?: string };
  connectionState: ConnectionState;
  viewerCapabilities: string[];
  main: ReactNode;
  side?: ReactNode;
  mobileTabs?: Array<{ key: string; label: string; icon: LucideIcon }>;
  bottomAction?: ReactNode;
};
```

**状态**

- Loading：阶段骨架、固定主操作占位。
- Ready：完整投影。
- Paused：只读内容 + 暂停说明。
- Reconnecting/Offline：保留最后确认投影，禁用 D2-D4 写操作。
- Incident frozen：停止行动，显示支持入口和事故说明。
- Ended/Read-only：保留结果与复盘入口。

**事件**

- `onOpenSettings`、`onOpenMembers`、`onReport`、`onRequestHelp`、`onRetryConnection`。
- WebSocket 状态变化只更新连接条，不卸载当前合法内容。

**动画**

- 首屏 320ms 分区域进入；不逐卡重复。
- 阶段切换由 GamePhaseIndicator + 主区内容切换共同完成。
- 侧面板和移动 Tab 使用 Motion Presence；reduced motion 使用 120ms 透明度切换。

**数据来源**

- `GET /api/v1/rooms/{room_id}`
- `GET /api/v1/games/{game_id}/projection`
- WS `room.*`
- WS `game.phase_changed`、`game.state_updated`、`game.connection_recovered`、`game.incident_frozen`

### 4.3 PlayerCard

**职责**

呈现公共玩家身份、席位、存活状态、在线状态、发言、投票和托管。任何角色/阵营数据只在授权视图中出现。

**Props**

```ts
type PlayerCardProps = {
  playerId: string;
  seatNo: number;
  displayName: string;
  avatarUrl?: string;
  status: "active" | "dead" | "eliminated" | "left" | "managed";
  connection: "online" | "offline" | "reconnecting" | "ai";
  speaking?: boolean;
  voteState?: "none" | "submitted" | "locked" | "not-eligible";
  roleHint?: { label: string; scope: "public" | "role" | "role_group" };
  selected?: boolean;
  disabledReason?: string;
  size?: "seat" | "list" | "compact";
};
```

**状态**

- Active、Dead/Eliminated、Disconnected、Managed、Speaking、Selected、Disabled、Unknown。
- 未授权角色显示为空，不显示模糊头像/边框暗示。

**事件**

- `onSelect`、`onInspectVote`、`onReport`。
- 夜间目标选择与普通查看使用不同视觉层，避免误触提交。

**动画**

- Speaking 使用小范围外环增强，120-180ms。
- Dead 状态通过一次确认后的饱和度/线条变化持久显示；不弹全屏骷髅。
- Vote submitted 使用短促金属压印反馈，260ms。

**数据来源**

- `gameProjection.players`
- `GET /api/v1/games/{game_id}/murder/me`
- `GET /api/v1/games/{game_id}/werewolf/me`
- WS `game.player_eliminated`、`game.vote_submitted`

### 4.4 CharacterCard

**职责**

呈现当前玩家的角色、身份、阵营、目标、秘密与已知关系；作者角色预览复用同一结构但使用编辑视图。

**Props**

```ts
type CharacterCardProps = {
  roleId: string;
  displayName: string;
  portraitUrl?: string;
  publicProfile?: string;
  factionLabel?: string;
  goals: Array<{ id: string; text: string; visibility: VisibilityScope }>;
  privateScript?: string;
  knownRelationships?: Array<{ roleId: string; label: string; knownFacts: string[] }>;
  revealState: "sealed" | "revealed" | "author-preview";
  searchQuota?: { used: number; remaining: number };
};
```

**状态**

- Sealed：只显示“身份尚未揭示”。
- Revealed：角色名、阵营、目标、私人内容。
- AuthorPreview：显示角色视角标识。
- Loading：稳定尺寸的封印面，不闪出角色名。

**事件**

- `onOpenFullScript`、`onMarkReadingComplete`、`onInspectRelationship`。
- 私密内容不进入全局搜索、Toast 或分享卡片。

**动画**

- Unseal 500-800ms；先确认手势，再打开封印。
- 角色名与目标先出现，私密正文后出现。
- Skip 立即到最终可读状态；中断时只移除临时遮罩，不改变权威身份。

**数据来源**

- `GET /api/v1/games/{game_id}/murder/me`
- `GET /api/v1/games/{game_id}/werewolf/me`
- `POST /api/v1/games/{game_id}/murder/reading-complete`
- `GET /api/v1/scripts/{script_id}/versions/{version_id}/role-preview`

### 4.5 ClueCard

**职责**

呈现已解锁线索的名称、正文、类型、来源、可见范围和关联事实。未解锁线索不得以标题轮廓或缩略图出现。

**Props**

```ts
type ClueCardProps = {
  clueId: string;
  title: string;
  body: string;
  clueType?: "physical" | "testimony" | "public" | "private";
  visibility: VisibilityScope;
  sourceLabel?: string;
  revealedAt?: string;
  imageUrl?: string;
  relatedKnownFacts?: Array<{ id: string; label: string }>;
  variant?: "dossier" | "testimony" | "physical" | "timeline";
  state?: "revealed" | "pinned" | "compared" | "claimed";
  actions?: Array<"inspect" | "compare" | "pin" | "askDm" | "archive" | "share">;
};
```

**状态**

- Revealed、Pinned、Compared、Claimed、Asset failed。
- `claimed` 只显示规则允许的“已被发现/不可重复”；不默认显示取得者。

**事件**

- `onInspect`、`onCompare`、`onPin`、`onAskDm`、`onArchive`。
- 获取成功由服务端 `game.clue_unlocked` 确认后触发；失败、被抢占、次数不足不播放获得动画。

**动画**

- 线索发现：聚焦 180ms → 卡面翻转 420-700ms → 正文阅读 → 收入档案 300ms。
- 比较：卡边标记与关系线 240-360ms。
- 归档后保留列表高亮与计数，不重复自动播放。

**数据来源**

- `GET /api/v1/games/{game_id}/murder/clues`
- `POST /api/v1/games/{game_id}/murder/searches`
- WS `game.clue_unlocked`
- `GET /api/v1/games/{game_id}/replay`

### 4.6 EvidencePanel

**职责**

管理已解锁线索档案、筛选、详情、比较、置顶、证据板和 DM 询问入口。它是档案工具，不应变成无边界 Modal。

**Props**

```ts
type EvidencePanelProps = {
  clues: ClueCardProps[];
  scopeFilter: "all_visible" | "public" | "private" | "mine";
  phaseKey: string;
  selectedClueIds: string[];
  relationshipHints?: Array<{
    fromClueId: string;
    toClueId: string;
    kind: "fact" | "hypothesis" | "contradiction";
  }>;
  layout: "list" | "board";
  canAskDm: boolean;
  canPin: boolean;
};
```

**状态**

- Loading：筛选栏与列表骨架。
- Empty：尚无发现，显示当前搜证范围与合法动作。
- Partial：部分线索加载或图片失败。
- Error：列表失败，保留已缓存线索。
- Permission denied：隐藏正文并给出返回安全页。
- Board fallback：列表视图始终可用。

**事件**

- `onFilter`、`onOpenClue`、`onCompare`、`onPin`、`onAskDm`、`onOpenBoard`。
- 证据板与列表共享选中状态；过滤不自动取消用户已选证据。

**动画**

- 列表项当前视口 Stagger 每项 30-50ms，总时长不超过 240ms。
- Pin 与 Compare 使用布局动画表达对象移动和关系建立。
- 关系图线只从真实 hint 生长；不创建装饰性随机连线。

**数据来源**

- `GET /api/v1/games/{game_id}/murder/clues`
- `M01` 的 `known_relationships`
- `G02` 事件增量
- WS `game.clue_unlocked`

### 4.7 RelationshipGraph

**职责**

呈现人物、证据、阵营或事件之间的授权关系。视觉连线必须对应已知事实、假设或矛盾，不能装饰性生成。

**Props**

```ts
type RelationshipGraphNode = {
  id: string;
  kind: "character" | "clue" | "event" | "faction";
  label: string;
  visibility: VisibilityScope;
  state?: "known" | "unknown" | "contradiction" | "focus";
};

type RelationshipGraphEdge = {
  id: string;
  source: string;
  target: string;
  label?: string;
  kind: "fact" | "hypothesis" | "contradiction";
  directed?: boolean;
  visibility: VisibilityScope;
};

type RelationshipGraphProps = {
  nodes: RelationshipGraphNode[];
  edges: RelationshipGraphEdge[];
  focusNodeId?: string;
  editable?: boolean;
  compact?: boolean;
  onSelectNode?: (id: string) => void;
  onSelectEdge?: (id: string) => void;
};
```

**状态**

- Loading、Empty、Ready、Filtered、Focused、Unavailable。
- 未知节点只表示“尚未发现的关系”，不暴露实际数量或位置。
- Canvas/SVG 必须有列表/邻接表替代。

**事件**

- `onSelectNode`、`onSelectEdge`、`onToggleKind`、`onOpenAccessibleList`。
- 编辑器可新增/删除关系；游戏页永远只读或提交结构化意图。

**动画**

- 图进入使用节点 180ms、边 220-360ms，不持续呼吸。
- New relation 沿真实方向生成 320ms。
- Focus 使用局部缩放与邻接高亮，避免全图镜头移动导致眩晕。

**数据来源**

- `M01.known_relationships`
- `GET /scripts/{id}/versions/{version_id}/editor`
- `GET /scripts/{id}/versions/{version_id}/role-preview`
- `GET /games/{id}/replay`

### 4.8 Timeline

**职责**

按服务端时间与事件序展示创作时间线、阶段时间线、行为链或复盘事件。创作时间线与对局事件必须区分来源。

**Props**

```ts
type TimelineItem = {
  id: string;
  seq?: number;
  occurredAt?: string;
  relativeOrder?: number;
  title: string;
  summary?: string;
  kind: "canon" | "phase" | "action" | "reveal" | "vote" | "world";
  visibility: VisibilityScope;
  sourceLabel?: string;
  issueCount?: number;
};

type TimelineProps = {
  items: TimelineItem[];
  activeId?: string;
  mode: "editor" | "game" | "replay";
  groupBy?: "phase" | "day" | "scene" | "none";
  onSelect?: (id: string) => void;
  onLoadMore?: () => void;
};
```

**状态**

- Loading、Empty、Partial、Ready、Selected、Data anomaly。
- 事件缺失必须标记“数据结构异常”，不推测补全。

**事件**

- `onSelect`、`onGroupChange`、`onLoadMore`、`onReportAnomaly`。

**动画**

- 新确认事件插入 220ms；当前阶段节点保持不动。
- 回放逐事件播放只在高亮与详情区域变化，不自动滚动劫持。

**数据来源**

- Editor：`S13`
- Game：`G01`、`G02`
- Replay：`G07`
- WS `game.phase_changed`、`game.state_updated`

### 4.9 NPCDialogue

**职责**

提供 NPC 身份、在场状态、公开/私密对话范围、历史、输入和结构化世界动作确认。

**Props**

```ts
type NPCDialogueProps = {
  npc: {
    npcId: string;
    displayName: string;
    portraitUrl?: string;
    present: boolean;
    publicProfile?: string;
    relationshipLabel?: string;
  };
  messages: Array<{
    id: string;
    speaker: "player" | "npc" | "system";
    text: string;
    visibility: VisibilityScope;
    safetyStatus?: "passed" | "rewritten" | "blocked";
  }>;
  conversationVisibility: "private" | "scene";
  aiState: "idle" | "thinking" | "streaming" | "degraded" | "failed";
  suggestedActions?: Array<{ actionId: string; label: string; requiresConfirmation: boolean }>;
  canSend: boolean;
};
```

**状态**

- Ready、NPC absent、Busy、Private、Scene-public、Waiting、Streaming、Safety replaced、Relationship changed、Ended。
- `safety replaced` 使用模板说明，不展示内部安全原因或隐藏知识。

**事件**

- `onSend`、`onConfirmWorldAction`、`onEndConversation`、`onShowItem`、`onRetry`。
- AI 文本流只能作为临时展示；正式 NPC 台词由最终授权消息持久化。

**动画**

- NPC 回应使用 120ms 气泡出现；逐字呈现只在安全终稿之后或受控流式分段。
- 用户阅读长文本时不持续显示循环打字点；超时显示“正在回应”。

**数据来源**

- `GET /games/{game_id}/rpg/npcs/{npc_id}/dialogues`
- `POST /games/{game_id}/rpg/npcs/{npc_id}/dialogues`
- `P05` 世界动作
- WS `npc.dialogue_generated`、`ai.stream.chunk`、`ai.run.degraded`

### 4.10 StoryScene

**职责**

承载环境、构图、剧情叙事、热点、场景状态和局部氛围，不把整个场地包装成一张圆角卡。

**Props**

```ts
type StorySceneProps = {
  scene: {
    id: string;
    title: string;
    description: string;
    timeLabel?: string;
    weather?: "clear" | "rain" | "fog" | "storm" | "snow";
    phaseKey: string;
  };
  assets: {
    backgroundUrl?: string;
    backgroundMobileUrl?: string;
    foregroundUrl?: string;
    dominantColor?: string;
  };
  hotspots?: Array<{
    id: string;
    label: string;
    kind: "search" | "npc" | "exit" | "object";
    state: "available" | "pending" | "discovered" | "claimed" | "unavailable";
    x: number;
    y: number;
    disabledReason?: string;
  }>;
  actionable: boolean;
  onHotspot?: (id: string) => void;
};
```

**状态**

- Loading asset、Ready、Asset failed、Low-performance fallback、Phase changed、Scene changed。
- Asset failed 使用设计好的色块/纸面/场所标题，不出现 broken image。

**事件**

- `onHotspot`、`onOpenHotspotList`、`onRetryAsset`。
- 热点必须有按钮语义和列表替代。

**动画**

- 场景进入 420ms，局部光 280ms；不做滚动视差。
- 场景切换使用方向性遮罩或叠化，reduced motion 使用短切/透明度。
- 背景动画不覆盖文本或主操作区域。

**数据来源**

- `GET /games/{game_id}/projection`
- `GET /games/{game_id}/murder/search-points`
- `GET /games/{game_id}/rpg/context`
- WS `game.phase_changed`、`game.state_updated`

### 4.11 GamePhaseIndicator

**职责**

常驻显示模式、阶段、轮次、剩余时间、暂停、连接和阶段合法性。它是运营层核心，不以装饰为中心。

**Props**

```ts
type GamePhaseIndicatorProps = {
  mode: "script_murder" | "werewolf" | "rpg";
  phaseKey: string;
  phaseLabel: string;
  roundNo?: number;
  serverDeadlineAt?: string;
  serverTimeOffsetMs: number;
  paused?: boolean;
  connectionState: ConnectionState;
  waitingForCount?: { submitted: number; required: number };
  compact?: boolean;
  onOpenPhaseHelp?: () => void;
};
```

**状态**

- Running、Warning、Expired pending authority、Paused、Reconnecting、Degraded、Offline、Frozen。
- 客户端计时到 0 只显示“等待服务端推进”，不得自行切换阶段。

**事件**

- `onOpenPhaseHelp`、`onPauseRequest`、`onRetryConnection`。

**动画**

- 阶段标签切换 260ms；倒计时数字不滚动动画。
- 进入预警区间使用一次状态过渡，不循环闪烁。

**数据来源**

- `gameProjection.phase`
- WS `game.phase_changed`、`game.timer_updated`
- `G03` 重连响应

### 4.12 VotingPanel

**职责**

呈现候选、本人选择、弃权、锁定、进度和规则允许的结果揭示。

**Props**

```ts
type VotingPanelProps = {
  roundKey: string;
  candidates: Array<{
    playerId: string;
    roleId?: string;
    displayName: string;
    seatNo?: number;
    eligible: boolean;
  }>;
  myVote?: { targetPlayerId?: string; abstain: boolean; locked: boolean };
  progress: { submitted: number; required: number };
  revealMode: "hidden" | "count_only" | "detail";
  tally?: Array<{ targetPlayerId: string; count: number }>;
  canChangeVote: boolean;
  confirmationRequired: boolean;
  disabledReason?: string;
};
```

**状态**

- Loading、Open、Selected、Confirming、Submitting、Submitted、Changeable、Locked、Closed、Resolved、No voting right。
- 投票成功必须由 REST/WS 权威确认，不提前显示票型。

**事件**

- `onSelectCandidate`、`onAbstain`、`onConfirm`、`onChangeVote`、`onInspectRules`。

**动画**

- 选中使用 140ms；确认按钮使用压印 180ms。
- 结果揭示只在 `game.vote_resolved` 后开始，400-900ms 按真实票数增长。
- 平票、弃权、无效票使用文字和形状，不只红/灰颜色。

**数据来源**

- `GET /games/{game_id}/murder/votes`
- `POST /games/{game_id}/murder/votes`
- `POST /games/{game_id}/werewolf/votes`
- WS `game.vote_submitted`、`game.vote_resolved`

### 4.13 MapExplorer

**职责**

呈现 RPG 地图节点、连接、当前地点、已发现、未知、阻断和可达状态，并提供键盘/列表导航。

**Props**

```ts
type MapExplorerProps = {
  currentSceneId: string;
  nodes: Array<{
    id: string;
    label: string;
    x: number;
    y: number;
    state: "current" | "discovered" | "reachable" | "unknown" | "blocked";
    sceneType?: string;
  }>;
  edges: Array<{
    id: string;
    from: string;
    to: string;
    state: "visible" | "unknown" | "blocked";
    label?: string;
  }>;
  previewSceneId?: string;
  compact?: boolean;
  onPreviewScene: (id: string) => void;
  onConfirmMove?: (id: string) => void;
};
```

**状态**

- Ready、No map data、Focused、Preview、Blocked、Stale scene version、Map load failed。

**事件**

- `onPreviewScene`、`onConfirmMove`、`onOpenAccessibleRoutes`、`onClosePreview`。
- 点击节点先预览目的地、代价和可达条件；世界变更需确认。

**动画**

- 当前节点使用静态高亮；可达路线不持续闪烁。
- 移动确认后，只在权威场景事件到达后切换地图与场景。
- 关系边进入 220ms，reduced motion 直接显示。

**数据来源**

- `GET /games/{game_id}/rpg/context`
- `POST /games/{game_id}/rpg/moves`
- WS `game.state_updated`
- `P04/P05`

### 4.14 QuestPanel

**职责**

呈现任务、目标进度、可见范围、时限、奖励/失败摘要与接受/放弃动作。

**Props**

```ts
type QuestPanelProps = {
  tasks: Array<{
    id: string;
    title: string;
    status: "locked" | "available" | "active" | "completed" | "failed" | "expired";
    description?: string;
    progress?: { current: number; target: number };
    visibility: VisibilityScope;
    expiresAt?: string;
  }>;
  activeTaskId?: string;
  compact?: boolean;
  onSelect: (id: string) => void;
  onAccept?: (id: string) => void;
  onAbandon?: (id: string) => void;
};
```

**状态**

- Empty、Loading、Available、Active、Completed、Failed、Expired、Permission denied。
- 空任务不渲染空白卡，显示探索/对话的合法下一步。

**事件**

- `onSelect`、`onAccept`、`onAbandon`、`onOpenSceneHint`。

**动画**

- 进度变化 220ms；完成使用一次确认标记，不循环庆祝。
- 任务更新保留原位置，避免队伍交流时面板跳动。

**数据来源**

- `GET /games/{game_id}/rpg/tasks`
- `POST /games/{game_id}/rpg/tasks/{task_id}/actions`
- WS `game.state_updated`

### 4.15 GenerationWorkflowPanel

**职责**

展示 LangGraph 生成工作流的当前节点、已完成、失败、并行分支、重试、人工审核、最终产物与安全操作。它不是单个 Loading 圆环。

**Props**

```ts
type WorkflowStepStatus =
  | "pending"
  | "running"
  | "succeeded"
  | "failed"
  | "skipped"
  | "waiting_human"
  | "cancelled";

type GenerationWorkflowPanelProps = {
  job: {
    jobId: string;
    status: "queued" | "running" | "waiting_human" | "partial" | "succeeded" | "failed" | "cancelled";
    progressBp: number;
    currentNode?: string;
    steps: Array<{
      stepKey: string;
      title: string;
      nodeType: "parse" | "model" | "validate" | "human_review";
      status: WorkflowStepStatus;
      progressBp?: number;
      attemptNo?: number;
      maxAttempts?: number;
      errorCode?: string;
      artifactSummary?: Record<string, number>;
      parallelGroup?: string;
    }>;
    validation?: { blockingCount: number; warningCount: number };
    artifactSummary?: Record<string, number>;
    error?: { code: string; summary: string; retryable: boolean };
  };
  lastEventId?: string;
  onRetryStep?: (stepKey: string) => void;
  onCancel?: () => void;
  onOpenIssue?: (stepKey: string) => void;
  onEnterEditor?: () => void;
};
```

**LangGraph 节点展示映射**

| Node | 用户文案 | 展示 |
| --- | --- | --- |
| `InputNode` | 校验素材与目标 | 参数摘要、权利声明 |
| `ParsingNode` | 解析原文 | 分块进度、来源定位、可读性问题 |
| `StructuringNode` | 建立故事圣经 | 人物/事件/地点/时间线候选数 |
| `CharacterNode` | 生成角色与关系 | 与 SceneNode 并行分支 |
| `SceneNode` | 生成场景与地图 | 与 CharacterNode 并行分支 |
| `ClueNode` | 生成线索与搜证点 | 线索、任务、关联事实摘要 |
| `PlotNode` | 编排剧情与阶段 | 分支、阶段、结局可达性 |
| `ImagePromptNode` | 生成图片描述 | 不阻塞文本产物；可显示安全 Prompt 状态 |
| `ImageGenerationNode` | 生成图片 | 可降级为占位，明确异步补齐 |
| `ValidationNode` | 一致性与可玩性校验 | 阻断/警告/信息三级 |
| `RepairNode` | 自动修复 | 修复轮次、问题减少情况 |
| `HumanReviewNode` | 等待人工确认 | 差异、锁定字段、批准/修复/拒绝 |
| `PublishNode` | 生成发布候选 | 候选包、校验报告、进入编辑器 |

**状态**

- queued：全部 pending，显示真实排队位置或“等待 worker”。
- running：当前节点 running，并行分支独立展示。
- waiting_human：不伪造百分比，显示人工审核节点。
- partial：成功产物可复用，失败片段可定位。
- succeeded：显示最终产物摘要与进入编辑器。
- failed：保留已完成节点，失败节点可重试或转人工。
- cancelled：已丢弃的候选不进入草稿。
- SSE disconnected：最后事件继续可见，显示“仍在处理/等待重连”。

**事件**

- `onRetryStep`、`onCancel`、`onOpenIssue`、`onEnterEditor`、`onDownloadReport`。
- `retry` 必须携带 job/stage/idempotency key。

**动画**

- 节点状态由 SSE 事件驱动；边线只在状态确认后推进。
- 当前节点 Progress 使用 transform，避免 width 过渡造成布局抖动。
- 失败节点使用 260ms 金属状态切换；禁止循环红闪。
- reduced motion 下直接切换状态，保留文本、错误码和重试按钮。

**数据来源**

- `GET /api/v1/generation-jobs/{job_id}`
- SSE `GET /api/v1/generation-jobs/{job_id}/events`
- `POST /api/v1/generation-jobs/{job_id}/actions`

### 4.16 其他复用组件

| 组件 | 用途 | 关键要求 |
| --- | --- | --- |
| `AppShell` | 全局导航、网络状态、通知 | 游戏页可切换为全屏壳 |
| `ScriptCard` | 剧本列表/推荐 | 稳定封面比例、来源、AI 比例、可用状态 |
| `RoomCard` | 大厅房间 | 人数、版本、可见性、快速加入 |
| `ObjectTree` | 编辑器对象导航 | 键盘可达、问题徽标、懒加载 |
| `IssueList` | 校验问题 | 阻断/警告/信息、对象定位、来源 |
| `VersionBadge` | 版本与发布状态 | 不可变版本醒目，旧版本提示 |
| `ConnectionBanner` | 全局网络/重连 | 保留最后确认状态，不阻塞阅读 |
| `PrivacySeal` | 私密范围 | 图标 + 文本 + 持久状态 |
| `CinematicOverlay` | 剧情揭示 | 可跳过、可中断、焦点管理 |
| `AudioSettings` | 声音与动效偏好 | 持久设置、立即生效 |
| `EmptyState` | 空内容 | 说明原因 + 合法下一步 |
| `ErrorPanel` | 可恢复错误 | 后果、requestId、重试/返回 |
| `OfflineNotice` | 离线 | 缓存可读、写操作禁用 |

---

## 5. 页面布局与交互

### 5.0 全局页面壳与路由

#### Global App Shell

```text
┌────────────────────────────────────────────────────────────────────┐
│ Logo / 首页 / 大厅 / 社区 / 工坊 / 搜索 / 通知 / 个人菜单        │ 64px
├────────────────────────────────────────────────────────────────────┤
│ Optional global connection / maintenance / content notice           │ 36px
├────────────────────────────────────────────────────────────────────┤
│ Page content                                                        │
│ max-width 1600px for editorial pages                                │
│ edge-to-edge for gameplay / editor / admin                          │
└────────────────────────────────────────────────────────────────────┘
```

- 顶部导航清晰、克制，不以超大 Logo 占据空间。
- 当前模式/房间/阶段在对应页面壳内持续可见。
- 全局网络条只在连接异常、维护或缓存模式出现。
- 对局页离开路由时若有未完成关键行为，使用明确后果 Dialog；普通阅读不滥用拦截。

#### 路由与页面覆盖

| Route | 页面 | 设计章节 |
| --- | --- | --- |
| `/home` | 首页 | 5.1 |
| `/auth` | 登录注册 | 5.2 |
| `/lobby` | 游戏大厅 | 5.3 |
| `/scripts/:scriptId` | 剧本详情 | 5.4 |
| `/community` | 内容社区 | 5.18 |
| `/workshop` | AI 剧本工坊 | 5.5 |
| `/workshop/import` | 文档导入/创意输入 | 5.5 |
| `/workshop/:jobId/progress` | 生成进度 | 5.6 |
| `/workshop/:draftId/edit` | 剧本编辑器 | 5.7 |
| `/rooms/create` | 创建房间 | 5.8 |
| `/rooms/:roomId/waiting` | 房间等待 | 5.9 |
| `/play/murder/:roomId` | 剧本杀 | 5.10 |
| `/play/werewolf/:roomId` | 狼人杀 | 5.11 |
| `/play/rpg/:roomId` | RPG | 5.12 |
| `/play/:roomId/npc/:npcId` | NPC 对话 | 5.13 |
| `/play/:roomId/result` | 结算 | 5.14 |
| `/play/:roomId/replay` | 复盘 | 5.15 |
| `/me` | 个人中心 | 5.16 |
| `/me/history` | 游戏历史 | 5.17 |
| `/me/scripts` | 我的剧本 | 5.16 |
| `/me/achievements` | 成就 | 5.16 |
| `/me/friends` | 好友 | 5.16 |
| `/admin/*` | 运营后台 | 5.19 |

### 5.1 首页 `/home`

**体验简报**

- 用户：游客、回访玩家、创作者。
- 当前问题：现在能玩什么，如何最快开始或继续？
- 主焦点：继续进行中房间（若有），否则三模式门槛。
- 必须常驻：模式可用性、登录状态、推荐内容来源。
- 不得出现：假活动、假热度、未授权最近记录、全屏营销 Hero。

**桌面布局**

```text
[continue-play strip, conditional]
[murder threshold] [werewolf threshold] [rpg threshold]
[recommended scripts: horizontal list]
[hot rooms: compact list] [recent play + community]
```

- 页面最大宽度 1600px，首屏下方露出推荐区标题。
- 三模式门槛不同构：
  - 剧本杀：倾斜的案卷封面、证据印章、当前热门案件。
  - 狼人杀：圆形席位局部、昼夜时间、当前公开对局数。
  - RPG：环境远景、地图边角、当前可加入世界。
- 模式面板不是三个同色卡片；每个面板使用自身场景裁剪与信息顺序。
- 推荐剧本保持统一卡片解剖，但允许封面材料随内容主题变化。
- 热门房间使用紧凑行，不使用装饰性大卡。

**移动布局**

1. 继续游玩。
2. 三模式纵向门槛。
3. 推荐剧本。
4. 热门房间。
5. 最近游玩与社区。

模式入口在 360x800 下必须完整显示模式名、可用状态和首要操作，不依赖 Hover。

**交互**

| 动作 | 合法状态 | 立即反馈 | 权威结果 | 失败恢复 |
| --- | --- | --- | --- | --- |
| 继续游玩 | 房间成员且进行中/等待 | 按钮 Loading | 校验最新房间状态后跳转 | 已结束 → 结果/历史 |
| 进入模式 | 模式可用 | 过滤大厅状态 | 跳转大厅并应用 mode | 未开放 → 禁用 + 阶段 |
| 打开推荐剧本 | 公开/授权可见 | 链接 Pending | 详情 DTO | 下架 → 不可用说明 |
| 快速加入热门房间 | 公开、可加入 | 匹配中 | 服务端席位 | 满房 → 保留其他房间 |
| 登录 | 游客 | 登录弹层/页面 | Session | returnTo 白名单 |

**状态**

- Loading：继续条、模式区、推荐区分别骨架；单区块失败不阻塞全页。
- Empty：无最近记录 → 三模式入口；无推荐 → 官方精选；无热门房间 → 剧本库。
- Error：个性化推荐失败使用公开热门数据。
- Offline：显示缓存公开内容与网络条；快速加入/继续写操作禁用。
- 未开放模式：展示“即将开放”与版本阶段。

**数据**

- `GET /api/v1/me?include=profile,privacy,recent_games`
- `GET /api/v1/scripts?sort=...`
- `GET /api/v1/rooms?status=waiting&joinable=true`
- 聚合推荐/官方精选接口在 API_DESIGN 中未明确，当前标记为契约缺口；未接入前只能使用公开搜索兜底。

### 5.2 登录注册 `/auth`

**体验简报**

- 当前问题：如何以最少步骤登录并返回原目标？
- 主焦点：登录/注册表单，不是平台宣传。
- 必须常驻：returnTo 来源、协议版本、登录方式。
- 不得出现：账号是否存在提示、外部任意 URL 跳转。

**桌面布局**

- 中间窄表单列 420-480px。
- 右侧仅在有邀请/进行中目标时显示上下文摘要；无上下文时保持留白或轻量三模式标记。
- 不使用大幅插画或营销文案。

**移动布局**

- 单列表单，顶部显示“登录后继续加入房间/创建内容”等目标提示。
- 输入框和提交按钮保持 44px 以上触控高度。

**交互与状态**

- 子状态：登录、注册、找回、验证码已发送、提交中、需要补充资料、账户受限。
- 表单错误就地显示；账号安全和限流错误不泄露账号存在性。
- 邀请上下文由服务端短期 token 保存；成功后自动恢复加入流程。
- Loading 只用按钮级状态，页面级骨架仅用会话恢复。

**数据**

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/logout`
- `POST /api/v1/invitations/resolve`

### 5.3 游戏大厅 `/lobby`

**体验简报**

- 当前问题：哪些内容适合我现在的人数、时长和偏好？
- 主焦点：筛选后的结果。
- 必须常驻：模式、筛选摘要、结果数量、排序。
- 不得出现：私人房间、未公开剧本、虚假可加入状态。

**桌面布局**

- 左侧筛选栏 240-280px，主区结果。
- 顶部模式 segmented + 搜索 + 排序。
- 结果支持网格/列表；房间与剧本有不同卡解剖，不混用字段。
- 切换筛选保留旧结果并局部显示 loading，不整页闪烁。

**移动布局**

- 顶部粘性搜索/排序条。
- 筛选使用全高 Drawer，显示已选摘要与清除。
- 结果列表优先于网格，避免窄屏卡片过大。

**URL 状态**

```text
/lobby?game_type=script_murder&min_players=6&duration_max=120&difficulty=standard&sort=recommended
```

- 非法筛选自动回退并提示。
- 刷新、分享保持筛选。

**交互**

| 动作 | 反馈 | 失败 |
| --- | --- | --- |
| 搜索/筛选 | 旧结果保留 + 局部 loading | 保留上次结果 |
| 快速加入 | 匹配租约状态，可停止 | 超时 → 继续搜索 |
| 打开详情 | 稳定卡片点击区 | 下架 → 不可用 |
| 复制房间码 | Toast + 可复制状态 | 需房间权限 |

**状态**

- Loading、Filtered、Empty、Error、Offline、QuickJoinPending、No Joinable Rooms。
- 私人房间不出现在结果。

**数据**

- `GET /api/v1/scripts`
- `GET /api/v1/rooms`
- `POST /api/v1/rooms/{room_id}/join`
- `POST /api/v1/invitations/resolve`

### 5.4 剧本详情 `/scripts/:scriptId`

**体验简报**

- 当前问题：这个剧本是否适合本局？
- 主焦点：剧本身份、人数、时长、难度与创建房间。
- 必须常驻：内容来源、版本、内容提示、主 CTA。
- 不得出现：私密角色本、完整真相、未审核评论正文。

**桌面布局**

- 顶部封面 + 元信息，最大 1600px。
- 下半部为无剧透简介、玩法说明、内容提示、评分短评、相似推荐。
- 主 CTA 固定在右侧栏或顶部 sticky 操作条，不覆盖正文。
- 作者视图显示编辑入口、校验状态与当前草稿，不与公开版本混淆。

**移动布局**

- 封面 16:9 或 4:5 裁剪保留主体。
- 元信息紧随封面。
- 主 CTA 固定底部，内容提示在 CTA 上方的可读位置。

**页面状态**

- Public、Private、Invite-only、Unlisted、Unavailable、Version updated、No permission。
- CTA 校验完成前文案为“检查可用性”，不可提前点击创建。
- 无评论时显示评分引导；封面缺失使用模式化封面 fallback。

**数据与交互**

- `GET /api/v1/scripts/{script_id}`
- `GET /api/v1/scripts/{script_id}/reviews`
- `GET /api/v1/scripts/{script_id}/comments`
- `PUT /api/v1/scripts/{script_id}/favorite`
- `POST /api/v1/share-links`
- 创建房间携带 `scriptVersionId`，进入 `/rooms/create`。

### 5.5 AI 剧本工坊 `/workshop`

**体验简报**

- 当前问题：用什么素材、目标配置和权利声明开始生成？
- 主焦点：当前步骤的输入和可提交摘要。
- 必须常驻：步骤进度、保存状态、内容来源与配额。
- 不得出现：上传未扫描完成即可启动、未声明权利的任务、伪造预计结果。

**三步结构**

| Step | 内容 | 完成条件 |
| --- | --- | --- |
| 1 素材来源 | 上传 DOCX/PDF/TXT/MD、粘贴文本、创意生成、空白模板 | 文档扫描 clean 或文本有效 |
| 2 目标配置 | 模式、人数、时长、难度、语言、风格、AI 补充开关 | 服务端参数 Schema 合法 |
| 3 权利与检查 | 原创/授权/公版/私人试玩、内容分级、成本/时长预估 | 声明确认 + 配额可用 |

**桌面布局**

```text
[left step nav 220] [form/main 720-880] [summary rail 320]
                     [draft list below]
```

- 左栏显示步骤、草稿入口、最近任务。
- 主区只展示当前步骤，不把所有配置塞入一张长表单。
- 右栏持续显示“任务摘要”，包括文件、模式、人数、预计时长。

**移动布局**

- 单列步骤。
- 文件上传与文本粘贴完整支持。
- 复杂模式配置进入独立全屏步骤，底部固定“下一步/开始生成”。

**上传交互**

1. 选择文件 → 显示格式、大小、哈希和配额检查。
2. 请求短期上传凭证。
3. 上传显示字节进度 + 扫描状态。
4. 确认上传后显示 `scan_status` / `parse_status`。
5. 只有 `scan_status=clean` 才可启动任务。

**状态**

- New、Draft、Uploading、Scanning、Config incomplete、Rights unconfirmed、Quota insufficient、Ready to start、Deleting。
- 上传失败可重试/续传；不能重复创建同一项目。
- 删除项目只影响草稿，不删除历史房间引用；使用确认 Dialog。

**数据**

- `POST /api/v1/scripts`
- `POST /api/v1/scripts/{script_id}/documents/upload-intents`
- `POST /api/v1/script-documents/{document_id}/complete`
- `POST /api/v1/scripts/{script_id}/generation-jobs`
- 草稿/任务列表无明确聚合接口，列为契约缺口。

### 5.6 剧本解析进度 `/workshop/:jobId/progress`

**体验简报**

- 当前问题：系统做到哪里，是否有失败或人工确认，能不能进入编辑器？
- 主焦点：`GenerationWorkflowPanel`。
- 必须常驻：总状态、当前节点、已完成/失败节点、总体进度、下一步。
- 不得出现：无差别 Loading 圆圈、假百分比、未脱敏原文/Prompt。

**桌面布局**

```text
[top job header: file / mode / status / elapsed / leave-safe notice]
[main workflow 720-900] [right issues/artifacts 360]
[bottom sticky action: retry / cancel / enter editor]
```

- 节点时间线竖向显示，并行分支并排缩进。
- 当前节点显示子步骤、权重、尝试次数。
- 已成功节点显示可查看的脱敏产物摘要。
- 失败节点显示错误码、是否可重试、可恢复检查点。
- HumanReviewNode 显示差异、问题、锁定字段和“批准/要求修复/拒绝”。
- 校验问题按阻断、警告、信息分组，并能跳到来源对象或解析片段。

**移动布局**

- 顶部状态卡 + 纵向节点时间。
- 问题列表和操作使用底部 Sheet。
- 取消/重试固定在安全区域上方，不与导航重叠。

**进度可视化规则**

- 进度由节点权重和 SSE 更新计算，不由动画自行递增。
- 当前节点最多一个；并行分支可同时 running。
- 已完成节点不会因后续失败自动消失。
- Repair 轮次显示为从 Validation → Repair → Validation 的回环路径。
- 任务完成显示“生成候选已就绪”，不显示“已发布”。
- 离开页面任务继续；返回时以 S08/S09 状态重建。

**状态**

- Queued、Running、Waiting Human、Partial、Validation Failed、Ready for Edit、Cancelled、Completed、Expired。
- 长时间无更新超过阈值显示“仍在处理”，不伪造新进度。

**数据**

- `GET /api/v1/generation-jobs/{job_id}`
- SSE `GET /api/v1/generation-jobs/{job_id}/events?last_event_id=...`
- `POST /api/v1/generation-jobs/{job_id}/actions`

### 5.7 剧本编辑器 `/workshop/:draftId/edit`

**体验简报**

- 当前问题：哪些对象需要修复，如何在不破坏来源与锁定的前提下发布？
- 主焦点：当前对象的结构化编辑。
- 必须常驻：版本、保存状态、校验状态、问题入口、发布操作。
- 不得出现：未保存覆盖、静默冲突、角色预览泄露。

**桌面四区**

```text
[top bar: version / save / validation / publish]
[object tree 240-280] [editor 640-900] [issues+source 320-380]
```

- Object Tree：人物、关系、场景、事件、线索、阶段、规则、地图、NPC、任务。
- 中间：字段表单、锁定开关、局部重生成、差异对比。
- 右侧：问题列表、原文来源、置信度、AI 生成标识。
- 顶部：版本选择、自动保存、运行校验、角色预览、发布。

**编辑交互**

- 对象树加载后按对象懒加载表单。
- 当前对象有未保存修改时切换对象给出保存/放弃/取消。
- 字段锁定使用明确锁图标；局部重生成请求显示锁定路径。
- AI 候选以 Diff Panel 呈现，逐项接受/拒绝。
- 校验问题跳转到对象和字段路径，并高亮来源。
- 角色预览打开 `角色视角` 预览条，当前编辑数据不自动发布。

**发布流程**

1. `运行发布前校验`。
2. 阻断问题 > 0：发布按钮禁用并解释。
3. 选择渠道：私人、邀请、公开。
4. 确认内容分级、版本不可变性与旧对局影响。
5. 提交发布；成功后显示 `publication_id` 和版本状态。
6. 失败保持原版本指针不变。

**工坊对象与预览区**

桌面端在结构编辑之外提供一组显式预览视图，不与表单字段混在同一面板：

| 预览 View | 内容 | 数据与约束 |
| --- | --- | --- |
| 人物关系 | 有向关系、公开/私密范围、来源、矛盾提示 | `RelationshipGraph`；只画已存在关系 |
| 时间线 | 创作事件顺序、绝对/相对时间、并行与不确定时间 | `Timeline`；区分 `script_events` 与运行事件 |
| 角色卡 | 角色公开身份、私人目标、秘密、已知关系、投票权 | 只在作者/角色视角下显示秘密 |
| 场景 | 场景类型、可达条件、出口、搜索点、当前预览场景 | `StoryScene` 只读模式与场景字段联动 |
| 图片 | `ImagePromptNode` / `ImageGenerationNode` 产物、尺寸、来源、审核状态 | 可接受/拒绝候选图；失败使用占位 |
| 线索 | 线索类型、来源、可见范围、揭示门控、关联事实 | 关联搜证点和可推导路径 |
| 任务 | 目标条件、奖励、失败后果、可见范围 | 编辑状态与校验问题联动 |
| 一致性 | 阻断/警告/信息、对象路径、来源、修复建议 | 复用校验报告，不自行推断结果 |
| 发布预览 | 角色视角、无剧透详情、房间配置摘要、发布渠道差异 | 通过 `S20` + 服务端发布校验 |

图片预览状态：

- `pending`：显示安全 Prompt/尺寸/来源，不展示未完成图。
- `ready`：显示候选图、裁切、审核状态和接受/拒绝。
- `rejected/failed`：显示原因类别与重新生成/占位路径。
- 图片接受只更新候选产物，不自动发布。
- 角色图和场景图必须保留来源对象与版本引用。

人工修复流程：

1. 从问题列表定位对象与来源。
2. 选择人工编辑或局部重生成。
3. 锁定不允许 AI 修改的字段。
4. 查看候选 Diff，接受/拒绝逐项变化。
5. 重新运行一致性检查。
6. 通过发布预览确认角色可见范围。
7. 满足阻断为 0 后发布。

**移动布局**

- 支持查看对象和轻量字段编辑。
- 复杂关系图、规则模板、阶段图、批量差异显示“请使用桌面端完成”。
- 使用只读预览和问题定位，不压缩成三栏。

**状态**

- Autosaving、Saved、Dirty、Validating、Validation passed、Blocked issues、Reviewing、Published、Read-only、Edited elsewhere。
- 保存冲突必须提供“重新加载 / 保留本地差异 / 复制本地内容”。

**数据**

- `GET /scripts/{script_id}/versions/{version_id}/editor`
- `POST /scripts/{script_id}/versions/{version_id}/objects/{object_type}`
- `PATCH /scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}`
- `POST /scripts/{script_id}/versions/{version_id}/regenerations`
- `POST /scripts/{script_id}/versions/{version_id}/validate`
- `POST /scripts/{script_id}/versions/{version_id}/publish`
- `GET /scripts/{script_id}/versions/{version_id}/role-preview`

### 5.8 游戏创建房间 `/rooms/create`

**体验简报**

- 当前问题：如何按剧本约束创建一局可开始的房间？
- 主焦点：房间配置与最终校验摘要。
- 必须常驻：剧本版本、人数范围、可见性、规则模板。
- 不得出现：低于最低人数仍可创建、密码明文、未确认内容提示。

**桌面布局**

- 左侧内容摘要：封面、模式、人数、时长、版本、内容提示。
- 右侧表单：房间名、可见性、密码、人数、规则模板、AI 席位（支持时）。
- 底部 sticky 校验栏：人数合法、版本可用、隐私确认、创建按钮。

**移动布局**

- 内容摘要折叠为顶部摘要。
- 表单单列，底部固定主操作。
- 规则模板使用 Drawer，不把长规则塞入 select popover。

**状态**

- Editable、Content unavailable、Player count mismatch、Submitting、Created、Quota exceeded。
- 服务端创建成功后进入等待页；重复提交返回同一房间。

**数据**

- `POST /api/v1/rooms`
- `GET /api/v1/scripts/{script_id}`
- `POST /api/v1/rooms/{room_id}/invitations`

### 5.9 房间等待 `/rooms/:roomId/waiting`

**体验简报**

- 当前问题：谁已就位，还差什么条件才能开局？
- 主焦点：席位与准备状态。
- 必须常驻：房间码/邀请、玩家状态、开始条件、聊天入口。
- 不得出现：房主可见他人角色秘密、客户端自行认定可开局。

**桌面布局**

```text
[room header: title / mode / visibility / invite copy]
[seat grid main 720-900] [chat + rules 360]
[sticky readiness summary + primary action]
```

- 席位卡显示昵称、席号、准备、在线、房主和 AI；
- 空席显示邀请行动和最低人数，不使用虚构玩家。
- 规则摘要、内容提示、暂停策略在右栏可查看。
- 房主操作与玩家准备动作分离，避免权力层级混淆。

**移动布局**

- 席位、聊天、规则使用顶部 Tab。
- 底栏固定“准备/取消准备”或房主“开始游戏”；开始条件未满足时显示原因。

**交互**

| 动作 | 条件 | 反馈 |
| --- | --- | --- |
| 准备 | 等待阶段玩家 | 立即 pending，服务端广播后持久 |
| 开始 | 房主 + 条件满足 | 锁定摘要 + 版本检查 |
| 踢出 | 房主 + 等待阶段 | 二次确认对象与原因 |
| 转移房主 | 房主 + 目标在线 | 确认后广播 |
| 复制邀请 | 房间成员 | 短期 share URL + 复制反馈 |

**状态**

- Waiting、Not full、Full not ready、Starting、Version locking、Host offline、Room closed、Kicked、Invite expired。
- 实时连接失败：保留席位，聊天和准备显示只读/重连。

**数据**

- `GET /api/v1/rooms/{room_id}`
- `POST /api/v1/rooms/{room_id}/ready`
- `POST /api/v1/rooms/{room_id}/start`
- `POST /api/v1/rooms/{room_id}/host-actions`
- `POST /api/v1/rooms/{room_id}/invitations`
- WS `room.player_joined`、`room.player_left`、`room.player_ready`、`room.host_transferred`、`room.closed`
- 等待房聊天按 ADR-006、R13/R14 和 `room_chat` 流实现；故障不得阻塞准备/开始。

---

### 5.10 剧本杀游戏页 `/play/murder/:roomId`

**体验简报**

- 当前问题：当前阶段是什么，我知道什么，能做什么？
- 主焦点：场景/叙事与当前阶段主操作。
- 必须常驻：阶段、计时、角色入口、线索入口、聊天入口、连接状态。
- 不得出现：他人私人剧本、未发现线索、未揭示真相、房主额外私密视图。

**桌面布局**

```text
[phase / timer / members / settings]
[main scene + narrative + action 760-980] [role card + clue archive + private chat 340-380]
[public discussion / question / DM input, sticky bottom]
```

- 场景为 edge-to-edge 工作区，不放入圆角卡。
- 主区顶部显示场景标题、时间和当前戏剧问题。
- 搜证点在场景内和“搜证点列表”中都能进入。
- 右栏固定宽度，内部滚动；不因线索数量增长挤压主场景。
- 右栏使用“角色 / 线索 / 关系 / 时间线”组；证据关系由 `RelationshipGraph` 与真实关联事实生成。
- 底部聊天只在规则允许阶段出现；私聊入口明确对象与范围。

**移动布局**

- 顶部固定阶段/计时。
- 中央可滚动场景或叙事。
- Tab：剧情、线索、聊天、角色。
- 当前阶段主操作固定在 Tab 上方或安全区底部。
- 搜证点通过热点与列表按钮进入，不依赖 Hover。

**阶段内容优先级**

| 阶段 | 主焦点 | 主操作 | 次操作 |
| --- | --- | --- | --- |
| 角色阅读 | 私人角色本 | 标记阅读完成 | 查看目标/关系 |
| 搜证 | 场景与可用点位 | 选择点位 | 查看剩余次数 |
| 讨论 | 公共发言与已知线索 | 发送/质询 | 请求 DM 提示 |
| 私聊 | 允许对象与范围 | 发送私密消息 | 返回公共讨论 |
| 投票 | 候选与确认 | 提交/改票 | 查看规则 |
| 结算中 | 锁定状态与结果入口 | 等待规则结果 | 返回历史 |

**交互**

| 动作 | 立即反馈 | 权威结果 | 失败/恢复 |
| --- | --- | --- | --- |
| 搜证 | 点位锁定、显示“搜查中” | `M04` + `game.clue_unlocked` | 次数不足/已被取得 → 刷新点位 |
| 完成阅读 | 按钮 pending | `M02` completed count | 阶段变化 → 同步 |
| 发送聊天 | 草稿保存 | `G05` event_id | 限流/禁言 → 保留草稿 |
| 质询 | 目标与可见范围确认 | `M06` | 目标/阶段非法 → 原因 |
| DM 提示 | 当前提示级别/次数 | `G08` | AI 降级 → 模板 |
| 投票 | 选中 -> 确认 | `M07` | 过期/版本冲突 → 刷新票型 |

**状态**

- Reading、Searching、Discussing、Private Chat、Voting、Settling、Paused、Reconnecting、Read-only、Terminated。
- 无公开线索：显示当前搜证范围，不显示空白错误。
- 无私人线索：说明“目前没有你的私人发现”。
- 被抢占：显示真实点位状态，不显示取得者（若规则隐藏）。
- DM 生成：局部“正在叙述”，不覆盖搜证/投票按钮。

**数据**

- `G01` 游戏投影
- `M01` 我的角色与阶段
- `M02` 阅读完成
- `M03/M04/M05` 搜证与线索
- `M06` 质询
- `M07/M08` 投票
- `G04/G05` 消息
- `G08` AI DM
- WS `game.phase_changed`、`game.clue_unlocked`、`game.message_received`、`game.vote_submitted`、`game.vote_resolved`

### 5.11 狼人杀游戏页 `/play/werewolf/:roomId`

**体验简报**

- 当前问题：现在是昼/夜哪一轮，我的合法私密行动是什么，公共状态发生了什么？
- 主焦点：席位环与当前发言/行动。
- 必须常驻：昼夜、轮次、计时、本人状态、发言队列、投票入口。
- 不得出现：完整身份表、他人技能结果、狼队信息给非狼人、内部 AI 推理。

**桌面布局**

```text
[day/night / round / timer / phase]
[public narrative band]
[seat ring center 600-840] [private action / identity / speaking / voting 340-380]
[public speech / quick input]
```

- 席位环是页面主构图，不是玩家卡网格。
- 席位只显示公共状态；未知身份不通过边框、头像色或动画泄露。
- 夜间中心场景降低亮度，但席位名、当前行动和计时保持高对比。
- 白天提高中性光与清晰度，发言队列成为主要社会信息。

**移动布局**

- 顶部固定昼夜/轮次/计时。
- 中央使用紧凑环或 3 列席位摘要，保留当前发言者与本人席位。
- 身份、夜间行动、投票使用底部 Sheet；当前 Sheet 不覆盖固定主操作。
- 公开发言区在白天优先，夜间自动收起。

**夜间行动**

1. 服务端返回 `can_act` 与合法目标。
2. 私密行动面板置于侧栏或 Sheet。
3. 选择目标只能从 `legal_targets` 生成。
4. 提交后显示“已提交，等待结算”，不显示技能结果。
5. 夜晚结算完成后才通过授权事件展示个人结果。

**白天交互**

- 发言顺序显示当前、下一位和队列，不显示倒计时每毫秒变化。
- 到本人发言时提供文本输入和剩余时间；非本人回合输入禁用并说明原因。
- AI 发言显示克制“思考中”状态，不展示推理链。
- 投票候选只包含规则允许成员；平票/PK 来自模板状态。

**技能与死亡**

- 技能触发必须有确认，明确目标、触发事件和不可逆性。
- 淘汰先更新公共存活状态；角色揭示只在规则授权的独立 Beat 出现。
- 死亡玩家保留旁观/遗言合法能力，但隐藏所有存活行动面板。

**状态**

- Waiting、Role reading、Night actions、Night resolving、Dawn、Last words、Discussion、Voting、Vote resolve、Skill chain、Settling、Ended。
- Night private → submitted → waiting；不显示成功命中。
- AI timeout：模板发言/无行动状态；不阻塞阶段。
- Incident frozen：停止操作，显示“正在诊断”，不展示猜测结果。

**数据**

- `G01` 游戏投影
- `W01` 身份与技能
- `W02/W03` 夜间上下文与行动
- `W04` 发言
- `W05` 投票
- `W06` 遗言
- `W07` 技能触发
- WS `game.phase_changed`、`game.player_eliminated`、`game.vote_resolved`、`game.skill_triggered`

### 5.12 RPG 游戏世界页 `/play/rpg/:roomId`

**体验简报**

- 当前问题：我在哪里，世界发生了什么，我能如何行动？
- 主焦点：场景主体、环境叙事与当前合法行动。
- 必须常驻：地点、世界时间、场景版本、队伍状态、行动输入。
- 不得出现：未发现出口、NPC 私有知识、未授权个人叙事进入公共频道。

**桌面布局**

```text
[world header: time / location / party presence / connection]
[map & routes 260-300] [scene + narrative + action input 680-900] [party / tasks / inventory 340-380]
[public world event strip, conditional]
```

- 地图可以是拓扑图或地形图，但路线、费用、可达条件可读。
- 中间场景是主视觉，叙事与行动输入有稳定阅读区域。
- 右栏任务、队友、背包、已见 NPC 使用明确 Tab 或折叠组。
- 公共世界变化与个人叙事采用不同视觉结构，不混在同一消息流。

**移动布局**

- 场景第一信号：顶部场景裁剪 + 地点/时间。
- 行动输入固定在场景下方，键盘弹出时不被遮挡。
- 地图、队伍、任务、背包、角色进入 labeled Drawer。
- NPC 对话可全屏，返回场景按钮固定。

**自由行动流程**

```text
输入文本
→ 理解中
→ 候选动作 / 需要澄清 / 不可执行
→ 用户确认结构化动作
→ 结算中
→ 权威状态变化
→ 叙事化结果
```

- 解析无状态变化；只有确认后的 `P05` 可改变世界。
- 低置信度不得自动猜测执行。
- 多目标、高风险、队伍共同变更动作必须确认。
- 结构化快捷动作始终存在，自由输入不是唯一路径。

**剧情选择与动态事件**

- 剧情选择以当前场景、可见选项、队伍确认和不可逆性为上下文。
- 多分支选择显示后果类别（关系、世界变量、任务、路线），不预告未授权结局。
- 动态公共事件与个人叙事分栏或分组呈现；公共变化保留队伍与事件来源。
- `P11` 仅提交合法 `choiceId`；结局与隐藏事件由世界引擎确认后进入结算/结局画面。
- 队伍共同决策显示确认状态与等待成员，不把本地选择当已执行。

**状态**

- Exploring、Dialogue、Quest、Conflict、Waiting for party、World advancing、Personal paused、Reconnecting。
- 场景版本过期：提示世界已变化，刷新合法动作。
- 资源竞争失败：提供替代路径或重试，不重复扣物品。
- AI 叙事失败：状态变化仍生效，以模板描述兜底。

**数据**

- `P01` 场景/地图/队伍/任务上下文
- `P02/P03` 移动与探索
- `P04/P05` 自由文本解析与结构化动作
- `P06/P07` NPC 对话
- `P08/P09` 任务
- `P10` 物品
- `P11` 剧情选择
- WS `game.state_updated`、`npc.dialogue_generated`、`ai.stream.chunk`

### 5.13 NPC 对话 `/play/:roomId/npc/:npcId`

**体验简报**

- 当前问题：我在和谁说话，谁能听见，我能要求世界改变什么？
- 主焦点：NPC 身份与对话内容。
- 必须常驻：NPC 在场状态、可见范围、场景上下文、返回场景。
- 不得出现：NPC 记忆原文、内部知识边界、模型推理、未授权秘密。

**桌面布局**

- 作为 RPG 世界页侧面板或独立大面板，保留左侧场景缩略与队伍状态。
- 顶部：NPC 名称、头像/立绘、在场、关系标签、可见范围。
- 中部：对话历史；消息来源标注玩家、NPC、系统。
- 底部：输入、发送、结束对话、合法结构化动作。

**移动布局**

- 全屏对话，顶部 NPC 身份，正文可滚动。
- 底部输入固定并避让软键盘。
- 返回场景按钮常驻，不用手势作为唯一退出。

**世界动作确认**

- 交易、给物、接受任务、攻击、承诺等以结构化 Action Chip 呈现。
- 点击后显示影响对象、物品/任务变化、可见范围。
- 确认后调用 `P05` /任务/物品接口，不由对话接口直接改状态。

**状态**

- Ready、NPC not present、Busy、Scene public、Private、Waiting、Streaming、Safety replaced、Relationship changed、Transaction pending、Ended。
- 流式文本失败保留正式历史与状态，不显示失败片段为事实。
- 同一 NPC 多人对话显示“在场可听/排队/仅你可见”。

**数据**

- `GET /games/{game_id}/rpg/npcs/{npc_id}/dialogues`
- `POST /games/{game_id}/rpg/npcs/{npc_id}/dialogues`
- `P05` 结构化世界动作
- `P08/P09` 任务
- `P10` 物品
- WS `npc.dialogue_generated`、`ai.run.degraded`

### 5.14 游戏结算 `/play/:roomId/result`

**体验简报**

- 当前问题：规则确认的结果是什么，为什么？
- 主焦点：胜负/结局，而不是 AI 长文。
- 必须常驻：结果状态、胜方/结局、触发条件、复盘入口。
- 不得出现：AI 点评改变结果、私聊全文、未授权个人信息。

**桌面布局**

- 顶部是模式化结果场景和结论标题。
- 中部显示触发条件、关键统计、角色/阵营/结局揭示。
- 下部显示参与者、投票表/任务变化/世界状态与分享卡片入口。
- AI 点评异步加载在次要区域，失败时隐藏而非空白卡。

**三模式重点**

| 模式 | 首屏结果 | 二级因果 |
| --- | --- | --- |
| 剧本杀 | 正确指认/阵营/个人目标、凶手揭示 | 票型、关键线索、时间线 |
| 狼人杀 | 阵营胜负 | 夜间行动链、投票、技能链、死亡 |
| RPG | 结局标题与触发条件 | 任务、世界变量、关系、队伍贡献 |

**移动布局**

- 结果标题与胜负第一屏。
- 关键因果折叠为章节，避免长文本压住复盘按钮。
- 分享/复盘/再开固定操作区。

**状态**

- Settling、Result ready、AI review pending、Replay failed、Aborted result、Permission expired。
- 规则结果优先；AI 点评未完成不影响阅读和分享。
- 无统计时仍显示规则结论；无 AI 点评不显示空白区域。

**数据**

- `GET /games/{game_id}/result`
- `GET /games/{game_id}/replay`
- `POST /share-links`
- WS `game.result_ready`、`game.ended`

### 5.15 游戏复盘 `/play/:roomId/replay`

**体验简报**

- 当前问题：关键因果链是什么，我当时的判断与真相如何不同？
- 主焦点：权威时间线与事件详情。
- 必须常驻：阶段/时间筛选、视角、AI 点评来源。
- 不得出现：未经授权私聊、未允许角色本原文、AI 叙述伪装成事实。

**桌面布局**

- 左：阶段/时间线 280-320。
- 中：事件详情、角色视角、行动链、投票/世界状态。
- 右：推理对照、AI 标注、引用事件。
- 顶部显示当前 `viewerRoleId`、可见范围、数据版本。

**移动布局**

- 顶部阶段切换。
- 事件以可折叠组呈现。
- 详情、视角、AI 点评分别使用 Tab 或 Sheet。
- 支持按事件逐步播放，禁止 scroll hijacking。

**事件详情**

- 事件类型、序号、服务端时间、来源、可见范围。
- 事实摘要与引用 ID。
- AI 点评标注“解释”而非“权威事实”。
- 缺失事件显示数据异常，不出于推测补内容。

**状态**

- Ready、AI annotations pending、Read-only、Permission denied、Version unsupported、Timeline partial。
- 切换角色视角重新请求，不缓存其他视角数据。

**数据**

- `GET /games/{game_id}/replay`
- `GET /games/{game_id}/events`
- `GET /ai-runs/{ai_run_id}`
- WS 不用于历史补齐；重连使用 `G03`/`G07`

### 5.16 个人中心 `/me`

**体验简报**

- 当前问题：我如何管理资料、隐私、好友、作品与成就？
- 主焦点：个人摘要与当前管理 Tab。
- 必须常驻：本人/他人视角、隐私范围、资料审核状态。
- 不得出现：私密草稿、他人可见的完整历史、未授权统计。

**布局**

- 顶部个人摘要：头像、昵称、公开编号、简介、等级。
- Tab：个人资料、隐私、好友、成就、我的剧本。
- 桌面使用页内水平 Tab；移动使用 segment + 单列。
- 战绩摘要只显示当前查看者有权看到的字段。

**我的剧本**

- 草稿、校验中、审核中、已发布、已下架过滤。
- 每个项目显示版本、问题数、最近任务、下一步。
- 新项目入口保持明显。

**好友**

- 好友列表、待处理申请、拉黑摘要。
- 搜索、发送申请、接受/拒绝/删除、拉黑。
- 过期申请显示结果与重试路径。

**成就**

- 无成就时不显示空徽章墙，改为获取路径。
- 不展示虚假进度或未实现成就。

**状态**

- Self edit、Public read-only、Profile moderation pending、Account restricted、Loading、Error、Privacy saved。

**数据**

- `GET/PATCH /api/v1/me`
- `GET /api/v1/users/{user_id}/public-profile`
- `GET /api/v1/friends`
- `POST /api/v1/friend-requests`
- `PATCH /api/v1/friendships/{friendship_id}`
- 成就与作品列表接口未在 API_DESIGN 中明确，列为契约缺口。

### 5.17 游戏历史 `/me/history`

**体验简报**

- 当前问题：过去对局中哪些需要继续、复盘或隐藏？
- 主焦点：按时间分组的记录列表。
- 必须常驻：筛选、对局状态、复盘权限。
- 不得出现：其他玩家私密角色内容、无权限记录。

**桌面布局**

- 顶部筛选：模式、剧本、时间、结果、状态。
- 主区按日期分组；行卡片显示模式、剧本、角色、结果、时长。
- 详情行内展开，完整信息进入复盘。

**移动布局**

- 筛选 Drawer。
- 行卡片突出结果与状态，次级信息折叠。
- 继续/复盘操作在行尾或行下方，44px 触控。

**状态**

- Normal list、Privacy restricted、Unlisted but replayable、In-progress、Expired、Loading、Empty、Error。
- 继续房间前实时校验状态，失败时改为只读结果或房间结束。
- 空历史显示可玩推荐与大厅入口。

**数据**

- `GET /api/v1/me?include=recent_games` 仅覆盖近期，不覆盖完整筛选历史。
- 页面需要 `GET /me/games` 或等价历史聚合接口；当前列为契约缺口。
- 复盘进入 `G07`。

### 5.18 内容社区 `/community`

**体验简报**

- 当前问题：有哪些公开剧本、作者和讨论值得发现？
- 主焦点：公开内容流与来源。
- 必须常驻：搜索、筛选、来源、剧透状态。
- 不得出现：私人内容、下架正文、默认展开剧透。

**桌面布局**

- 顶部专题/搜索/筛选。
- 主区内容流 + 侧栏官方精选/作者/标签。
- 评论和讨论支持“无剧透/含剧透”标签；含剧透默认折叠。
- 内容卡显示官方、原创、AI 辅助比例和版本更新时间。

**移动布局**

- 顶部搜索粘性。
- 内容流单列；筛选 Drawer。
- 剧透折叠按钮、评论审核状态和举报入口保持可触达。

**状态**

- Guest browse、Logged-in interaction、Moderation pending、Content unlisted、Spoiler collapsed、Empty、Error。
- 评分仅完成对局后可提交；重复评分返回当前结果。
- 评论 pending 不等于 published；UI 必须显示状态。

**数据**

- `GET /api/v1/scripts`
- `GET /scripts/{script_id}/reviews`
- `POST /scripts/{script_id}/reviews`
- `GET/POST /scripts/{script_id}/comments`
- `PUT /scripts/{script_id}/favorite`
- `POST /share-links`
- `POST /reports`

### 5.19 运营与审核后台 `/admin/*`

后台是独立的密集工作台，不使用玩家侧游戏视觉。使用 Paper Daylight + 金属系统色，重点是可扫描表格、筛选、审计和危险动作确认。

#### `/admin/content`

- 列表：内容、版本、模式、来源、AI 比例、审核状态、举报数、更新时间。
- 详情：公开预览、版本差异、校验报告、审核记录。
- 动作：通过、退回、下架、冻结发布；所有动作二次确认并显示审计字段。

#### `/admin/reports`

- 列表：举报类型、对象、状态、优先级、提交时间。
- 详情：定位具体消息/内容版本/事件 ID；默认不展示私密正文。
- 敏感查看需要申请审批，UI 明确访问理由与临时授权范围。

#### `/admin/generation`

- 任务队列、节点失败率、重试次数、成本、产物质量。
- 任务详情复用只读版 `GenerationWorkflowPanel`，增加审计字段。
- 不展示完整 Prompt 或私密原文。

#### `/admin/incidents`

- 按 `roomId/gameId/aiRunId/eventSeq` 串联状态、事件、投影和 AI 调用摘要。
- 事故动作：暂停、终止、冻结、恢复预案。
- 不直接改写进行中游戏状态。

**后台状态**

- Loading、Empty、Filtered、Audit approval required、Read-only、Action pending、Action failed、Permission denied。

**数据**

- 当前 `API_DESIGN.md` 未定义后台 API 契约。后台页面应等待独立运营 API，不复用玩家公共 API。

---

## 6. 剧情画面方案

### 6.1 通用场景模型

剧情画面必须由权威状态驱动：

```ts
type CinematicSceneState = {
  eventId: string;
  eventSequence: number;
  mode: "script_murder" | "werewolf" | "rpg";
  roomId: string;
  gameId: string;
  phaseKey: string;
  visibility: VisibilityScope;
  sceneId?: string;
  timeOfDay?: "dawn" | "day" | "dusk" | "night";
  weather?: "clear" | "rain" | "fog" | "storm" | "snow";
  focalEntityId?: string;
  publicActorIds: string[];
  authorizedPrivateActorIds: string[];
  discoveredEntityIds: string[];
  blockedRouteIds: string[];
  dangerLevel: 0 | 1 | 2 | 3;
  ambientIntensity: "quiet" | "tense" | "critical";
};
```

画面分层：

| Layer | 作用 | 典型内容 |
| --- | --- | --- |
| Ambient | 气氛 | 雾、雨、尘、反射光 |
| Background | 地点 | 建筑、山谷、船舱、广场 |
| Midground | 戏剧主体 | 证据、NPC、席位、路线、关键物件 |
| Foreground | 景深与遮挡 | 门框、雨幕、椅背、植被 |
| Light/Weather | 视线与阶段 | 月光、灯池、黎明、火光 |
| Information | 任务信息 | 标题、阶段、范围、动作 |
| Interaction | 合法交互 | 热点、按钮、目标选择 |

除剧情主体外，画面必须保留可读的操作区。关键文字不与雨、雾、粒子共用低对比区域；必要时使用局部暗背板或纸面层。

### 6.2 案件开始

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 游戏场景从近乎全黑的边缘展开，镜头落在案件发生的场所。桌面或地面上的一个异常物件成为视觉锚点，但不提前给出结论。 |
| 构图 | 中心偏下 1/3 放置异常物；上部保留房间结构与光线；右侧 340px 为角色/阶段入口；底部保留当前主操作。 |
| 主体 | 场景中的“不对劲”物件：停摆钟、翻倒椅子、湿脚印、未关闭的门，任选一个并绑定真实场景事实。 |
| 背景 | 同一场景的完整环境，而不是抽象背景；背景应说明时间、天气、地点。 |
| 光影 | 冷暖冲突：主光源从侧后方进入，异常物处于窄暖光；环境保持低频可见。 |
| UI 元素 | 模式、案件/房间标题、阶段、计时、内容提示、当前动作、隐私入口。 |
| 动画过程 | 0-420ms 场景从边缘遮罩展开；180-900ms 光池建立；700-1400ms 案件标题与地点出现；1200-2200ms 当前动作可操作。 |
| 触发时机 | 服务端确认进入开场/调查阶段后，由 `game.phase_changed` 或首屏投影触发；动画不决定阶段。 |
| 安全规则 | 不在任何公共层出现凶手、角色秘密、完整真相。 |
| Reduced Motion | 静态场景直接显示；标题与动作 120ms 淡入。 |

### 6.3 进入新场景

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 通过门、走廊、地图路线或空间遮挡完成地点转换，保留方向连续性。新场景第一眼显示一个可调查/可交互主体。 |
| 构图 | 前场景的边框或遮挡向侧边退出；新场景以斜线或门洞进入；场景标题位于左上，当前动作在底部或右侧固定。 |
| 主体 | 新地点的核心可交互物、NPC 或路线节点。 |
| 背景 | 新地点背景和相邻空间提示，未发现出口以缺失而非假图标处理。 |
| 光影 | 从旧场景色温过渡到新场景主光；保留 20-30% 连续色，避免突兀。 |
| UI 元素 | 场景标题、时间、天气、可达出口列表、搜证/探索动作、返回路径。 |
| 动画过程 | 0-180ms 锁定旧操作；120-720ms 门框/遮罩移动；500-1000ms 新背景与主光建立；800-1200ms 场景信息和合法动作出现。 |
| 触发时机 | RPG `game.state_updated` 或剧本杀场景切换事件确认后触发。 |
| 安全规则 | 新场景中未授权 NPC、出口或线索不进入 DOM、不进入过渡克隆层。 |
| Reduced Motion | 直接切换背景；标题与操作 120ms 更新，无空间位移。 |

### 6.4 NPC 登场

| 项目 | 设计 |
| --- | --- |
| 画面描述 | NPC 从场景景深或遮挡后进入可交互区域；立绘/半身像与环境融合，先建立“谁在这里”，再显示关系与话题。 |
| 构图 | NPC 位于画面右 1/3 或中心偏右；左侧保留场景和玩家位置；对话入口靠近 NPC 但固定底部。 |
| 主体 | NPC 立绘、表情/姿态、身份标记。 |
| 背景 | 当前真实场景，不用抽象渐变；NPC 阴影与光源方向一致。 |
| 光影 | NPC 面部有可读轮廓光；环境光与其阵营/情绪有弱关联，但不覆盖语义色。 |
| UI 元素 | NPC 名称、在场状态、关系标签、可见范围、对话入口、可提供的结构化动作。 |
| 动画过程 | 0-160ms 场景局部聚焦；120-500ms 立绘从景深进入；400-820ms 姓名与在场状态；700-1100ms 对话入口可用。 |
| 触发时机 | 服务端确认 NPC 与当前场景/视角可见后触发；AI 台词不是登场依据。 |
| 安全规则 | 不显示 NPC 私有记忆、隐藏目标、未授权真名。 |
| Reduced Motion | NPC 立绘直接出现，姓名与入口 120ms 淡入。 |

### 6.5 发现线索

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 场景局部聚焦到来源点位，证据从环境对象变为可读档案，并留下进入档案/时间线的持久状态。 |
| 构图 | 桌面以 480-560px 证据面居中；来源场景保留在背后低对比；档案入口/计数在外部边缘。移动使用近全高 Sheet。 |
| 主体 | 线索标题、正文/图像、来源、可见范围、关键词、关联事实。 |
| 背景 | 原场景局部，确保用户知道“从哪里发现”。 |
| 光影 | 局部暖光只作用于证据；其余区域降至 20-30% 可见度，文本使用稳定背板。 |
| UI 元素 | 来源、可见范围、关键词、关联线索、收入档案、查看线索、跳过动效。 |
| 动画过程 | 0-180ms 聚焦；120-420ms 证据进入；380-760ms 翻转；700-1030ms 正文与关键词；960-1160ms 关系提示；1080-1380ms 档案落位；1380-1500ms 控件恢复。 |
| 触发时机 | 仅在服务端确认 `M04` 或 `game.clue_unlocked` 后触发；被抢占/次数不足不播放。 |
| 安全规则 | 私密线索只进入本人层；不把私密标题写入公共 Archive 计数动画或 Toast。 |
| Reduced Motion | 无翻转/空间移动；证据面 120ms 出现，内容立即按阅读顺序可读，计数 120ms 高亮。 |

### 6.6 解锁秘密

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 加密卷宗、角色卡或关系节点通过封印解除变为可读内容。画面强调“权限变化”，不强调“炫技”。 |
| 构图 | 中央窄幅卷宗或侧边私密面板；封印占据上半部，内容在下方展开；背景保持当前游戏空间。 |
| 主体 | 角色秘密、私人目标、关系内幕或揭示后的公共事实。 |
| 背景 | 根据来源选择档案桌面、角色卡背景或关系板；不可使用公共场景展示私密全文。 |
| 光影 | 暖金属封印 → 打开后的纸面阅读光；禁用大面积紫色光。 |
| UI 元素 | 可见范围、角色/事实名称、来源、解锁时间、关联事件、后续合法动作。 |
| 动画过程 | 0-160ms 输入/确认锁定；120-420ms 封印压印；360-760ms 卷宗边缘打开；700-1050ms 标题与正文可读；950-1250ms 关系/阶段入口出现。 |
| 触发时机 | 服务端 reveal gate 打开且返回当前 viewer 私密投影后触发。 |
| 安全规则 | 不显示秘密的标题长度、角色数量或内容形状；跳过不改变揭示状态。 |
| Reduced Motion | 封印直接成为已解锁状态，正文 120ms 淡入；焦点到标题。 |

### 6.7 玩家死亡

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 公共席位/玩家状态发生明确变化，姓名仍可读，死亡/淘汰状态成为持久信息；角色是否揭示由规则独立决定。 |
| 构图 | 狼人杀以席位环局部为中心；剧本杀/RPG以场景中角色状态或结果条为中心；死亡公告靠近舞台但不遮住玩家名。 |
| 主体 | 玩家席位、死亡原因允许公开的摘要、遗言入口。 |
| 背景 | 当前场景保持可辨；夜间降低环境，不隐藏操作 UI。 |
| 光影 | 从环境光退到低饱和；不使用闪烁、爆裂或骷髅贴纸。 |
| UI 元素 | 姓名、席位、状态“已出局/死亡”、原因（若公开）、遗言入口、旁观/托管。 |
| 动画过程 | 0-180ms 当前阶段锁定；120-520ms 席位状态变化；480-900ms 死亡公告出现；800-1200ms 遗言/下一步入口。 |
| 触发时机 | 仅由 `game.player_eliminated` 或权威结果事件触发；AI 发言不能触发。 |
| 安全规则 | 淘汰动画不推断隐藏身份；若身份揭示是独立事件，必须使用独立 Beat。 |
| Reduced Motion | 席位状态直接更新；公告 120ms 淡入，不发散、不缩放。 |

### 6.8 投票结果

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 票数/票型按服务端确认的事件展开，先显示结果状态，再显示每个候选的真实计数与规则后果。 |
| 构图 | 桌面中央是候选与真实票型关系；席位/角色卡在周围；规则后果在下方。移动以结果摘要开头，逐项展开候选。 |
| 主体 | 候选、有效票、弃权、平票/PK、出局或真相结论。 |
| 背景 | 使用当前模式的工作台或席位环，不切换为通用统计仪表盘。 |
| 光影 | 结果确认使用中性金属高亮，危险/出局使用语义危险色。 |
| UI 元素 | round、submitted/required、票数、弃权、锁定状态、规则说明、下一步。 |
| 动画过程 | 0-180ms 关闭投票输入；120-520ms 计票条增长；420-900ms 平票/出局标注；800-1200ms 后续动作出现。 |
| 触发时机 | 仅在 `game.vote_resolved` 后触发；`game.vote_submitted` 只更新进度。 |
| 安全规则 | 不按客户端猜测生成结果；票型可见性由规则决定。 |
| Reduced Motion | 票数直接显示；结果标题与后续动作 120ms 更新。 |

### 6.9 真相揭晓

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 从最终结果回到权威事件链：先宣布规则结论，再重建时间线与证据/证词关系，最后对比玩家判断。 |
| 构图 | 上部 40% 为真相标题与结论；中部为时间线/证据关系；下部为个人推理对照和复盘入口。 |
| 主体 | 凶手/阵营/结局结论、关键事件、决定性证据。 |
| 背景 | 事件相关场景拼贴只作为低对比环境，不替代结构化时间线。 |
| 光影 | 场景光退为档案纸面与金属结论条；保留一处暖证据高亮。 |
| UI 元素 | 结果状态、触发条件、关键事件序号、证据来源、视角选择、复盘、分享。 |
| 动画过程 | 0-260ms 场景收束；200-900ms 结果标题；700-1800ms 时间线按事件顺序展开；1500-2400ms 证据与结论连接；2300-3000ms 复盘动作可用。 |
| 触发时机 | `GET /games/{game_id}/result` 已存在且 `game.result_ready` 到达后触发。 |
| 安全规则 | AI 点评不覆盖规则结果；私聊/私人剧本不因结算自动公开。 |
| Reduced Motion | 直接展示结果与结构化时间线；只做 120ms 区块淡入。 |

### 6.10 隐藏结局

| 项目 | 设计 |
| --- | --- |
| 画面描述 | 世界状态已发生不可逆变化，环境先显示后果，再显示结局标题。画面必须对应已配置结局条件，不生成无根据的世界改变。 |
| 构图 | 全宽环境变化为主；结局标题位于左上或下部可读区；队伍贡献与结局条件在结果层持续可见。 |
| 主体 | 被改变的世界要素：闸门关闭、城市警报、遗迹开启、势力关系破裂。 |
| 背景 | 原世界场景的变体，明确可对照“之前/之后”；移动保留同一地点主体裁剪。 |
| 光影 | 环境色随世界变量变化，但 Danger/Success 仍使用语义色。 |
| UI 元素 | 隐藏结局标题、触发条件、公共/个人变化、队伍贡献、复盘、下一世界/返回大厅。 |
| 动画过程 | 0-240ms 世界事件锁定；200-1100ms 环境变化；900-1800ms 公共状态更新；1500-2400ms 结局标题；2200-3200ms 结果与复盘动作。 |
| 触发时机 | `P11` 或权威世界事件确认结局条件后触发；AI 不得在条件未成立时叙述隐藏结局。 |
| 安全规则 | 个人结局与队伍结局分开投影；不显示其他玩家的私人分支条件。 |
| Reduced Motion | 直接切换世界状态图和结局标题；不使用大面积横移或粒子爆发。 |

### 6.11 剧情资产策略

- 场景背景使用实际内容/生成位图，文本和信息层由 DOM 渲染。
- 角色立绘与证据图使用稳定裁剪、透明背景或受控卡片遮罩。
- 场景动效使用 CSS 光/雾/雨层；不因装饰性效果引入 WebGL。
- 所有图片提供 AVIF/WebP、桌面与移动裁剪、低性能静态 fallback。
- 图片失败显示地点 + 场景标题 + 材料化色块，不显示破图。
- 任何关键揭示动画结束后，必须留下时间线节点、档案条目、席位状态、世界变量结果或结果页区块。

---

## 7. Motion Design

### 7.1 动画语法

Motion 只表达四类关系：

- Inspect：局部聚焦，帮助用户检查对象。
- Unseal：私密/隐藏信息变为可读。
- Thread：线索、人物、事件之间建立关系。
- Advance：房间、阶段、地点或世界前进。

如果动画不能表达上述之一，它只能是静音、低频、可关闭的氛围效果。

### 7.2 全局配置

```tsx
import { MotionConfig } from "motion/react";

export function MotionProvider({ children }: { children: React.ReactNode }) {
  return (
    <MotionConfig reducedMotion="user" transition={{ duration: 0.18 }}>
      {children}
    </MotionConfig>
  );
}
```

- 页面和剧情动画直接读取 `useReducedMotion()`。
- CSS 动效使用局部媒体查询，不全局删除所有状态反馈。
- Motion 的 `layout` 只用于用户理解对象移动，不用于普通 Hover。
- 动画临时数据与权威投影分离；动画 props 不写入服务端状态。

### 7.3 动画契约矩阵

| 动画 | 触发条件 | 时长 | 缓动 | 初始状态 | 最终状态 | 可跳过 | 中断规则 | Reduced Motion |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 页面转场 | 路由提交且目标页壳可用 | 300-420ms | `enter` + `standard` | 透明度 0，8px 位移 | 目标页可读、焦点可用 | 是，快速导航直接切页 | 新路由、权限失败、重连替换立即取消 | 120ms 透明度切换，无位移 |
| 卡片动画 | 首次进入视口、筛选后新对象 | 180-260ms，Stagger 30-50ms | `enter` | 0.98 缩放、透明度 0 | 完整卡面 | 无需单独跳过 | 过滤/排序中断后只保留当前布局 | 直接显示，不 Stagger |
| 线索翻转 | 已确认 `game.clue_unlocked` | 420-700ms | `enter` + `emphasis` | 背面封印、Y=90° | 正面正文可读 | 是，点击“跳过动效” | 事件纠正/路由离开立即终止并切终帧 | 无 3D 翻转，120ms 交叉淡化 |
| 证据解锁 | reveal gate 打开且私密投影返回 | 800-1200ms | `standard` + `emphasis` | 封印闭合、正文不可读 | 卷宗打开、正文可读 | 是 | 重连/权限变化取消临时层 | 直接打开，正文 120ms 淡入 |
| NPC 登场 | 服务端确认 NPC 在当前场景可见 | 500-1100ms | `enter` | 人物 12px 偏后、0 透明度 | 人物、姓名、入口可用 | 是 | 场景切换/NPC 离场立即重定向或移除 | 直接出现，姓名 120ms 淡入 |
| 文字逐字呈现 | 已通过安全检查的最终文本 | 18-28ms/字符，总长上限 800ms | `linear` | 文本区域已占位，字符不可见 | 全文本、焦点不跳动 | 是，点击/按键显示全文 | 新消息、路由离开、安全替换立即显示最终安全文本 | 直接显示全文本；不使用打字机 |
| 场景切换 | 服务端场景切换事件 | 800-1400ms | `standard` | 旧场景主光、门框/遮罩 | 新场景可读且主操作可用 | 是，快速切场景 | 快照纠正、重连、用户返回取消 | 120ms 切图/淡化，不发散、不位移 |
| 游戏阶段切换 | 权威 `game.phase_changed` | 1200-2400ms | `standard` | 旧阶段 UI、环境 | 新阶段标题、动作、侧栏已可用 | 是，阶段信息可用后立即跳过 | 阶段版本纠正立即切到正确终帧 | 阶段标题和合法动作 <120ms 更新 |
| 投票 | 选择本地反馈；结果由 `game.vote_resolved` 驱动 | 选择 140ms；结果 600-1200ms | `quiet` + `standard` | 未选/计数 0 | 锁定选择或真实票型 | 结果可跳过 | 关闭窗口/阶段纠正取消结果动画 | 直接显示当前选择/真实票数 |
| 结局揭晓 | 结果已存在且授权 | 2400-3200ms | `standard` + `emphasis` | 场景收束、结果不可读 | 结果、时间线、复盘入口可读 | 是，跳过到结果页 | 路由/权限/数据纠正立即切终帧 | 静态结果与时间线 120ms 淡入 |

### 7.4 页面转场

- 同层导航使用轻量内容转场；不把 Header 复制到动画克隆层。
- 进入游戏页时保留房间标题和模式作为方向线索；退出游戏不播放未确认结果。
- 路由切换前先由 Router 提交目标页；目标页可显示骨架，但当前主操作不能因动画延迟。
- 移动端避免横向大范围滑动；使用 8px 内方向位移 + 透明度。
- Reduced Motion 使用 120ms 透明度，焦点按目标页顺序移动。

### 7.5 卡片动画

允许：

- 首屏可见列表最多 6-8 项的短 Stagger。
- 筛选后新增/移动对象用布局动画。
- 卡片 Hover 只改变边框、亮度或轻微 `translateY(-1px)`，不得改变尺寸。
- 选中证据、置顶证据、任务进度可使用连续移动。

禁止：

- 每次 React render 重播 Enter。
- 每个 Section 统一 Fade-up。
- 全列表无限 Stagger。
- 卡片内多层 3D 倾斜。
- Hover 后内容被浮层遮挡。

### 7.6 线索翻转与证据解锁

完整序列按以下 Beat 执行：

1. Focus：场景局部降低竞争信息，保持主操作可见。
2. Acquire：证据从来源对象进入可读区域。
3. Flip/Unseal：只揭示已授权卡面。
4. Read：标题、事实、来源、关键词按阅读顺序出现。
5. Connect：只对真实关联事实生长关系线。
6. Archive：更新已确认档案计数和时间线节点。
7. Restore：解除控件锁定，焦点落到证据标题或下一点位。

业务绑定：

- 组件只在 `clue` 或 `reveal` 数据已进入 Zustand 投影后启动。
- `archive` 计数使用服务端新值，不使用动画临时计数。
- 动画结束后的回调只负责焦点、滚动和临时 UI 清理。
- 跳过立即切到“已解锁、已归档”的稳定终帧。

### 7.7 NPC 登场与对话

- NPC 登场由场景可见性驱动，不由 `npc.dialogue_generated` 首次文本驱动。
- 角色立绘位移小、时间短；姓名和关系标签比装饰性光效先可读。
- 对话文本若流式显示，文本区域必须先占位；输入框与场景主操作不能因 Token 增长重排。
- 流式文本必须使用独立 `ai_stream`；最终消息到达后以正式事件替换展示层。
- 对安全替换文本，直接显示替换结果并说明“该回应已调整”，不重播逐字效果。

### 7.8 场景与阶段切换

阶段切换必须同时完成两项：

1. 环境/场景变化提示“世界前进”。
2. Operational Layer 更新并显示新的合法动作。

顺序：

```text
权威新阶段已加载
→ Header/阶段文本更新
→ 环境层切换
→ 当前阶段主区与侧栏更新
→ 主操作恢复
→ 非关键氛围继续
```

- 阶段文本最迟 1.5s 内可读。
- 主操作只在权威阶段与能力可用后启用。
- 旧阶段私有面板在公共过渡层开始前卸载。
- 重连时直接从当前阶段快照构建，不重播旧转场。

### 7.9 文字逐字呈现

适用：

- 已通过安全检查的 DM/NPC 叙事。
- 场景描述和结局叙事。
- 不适用于计时、投票、规则码、按钮标签、错误信息。

规则：

- 中文按 grapheme/字符分段，避免拆开组合字符。
- 文本层先占位；不能把输入框和主操作不断向下推。
- 总动画不超过约 800ms；长文本后续段落直接显示或按段落切换。
- 用户点击、按空格、方向键或点击“显示全文”立即完成。
- `aria-live` 只宣告最终完整段落，不逐字播报。
- Reduced Motion 直接显示完整文本。

### 7.10 投票动画

- 选择候选人：140ms 边框/印章变化，不移动候选位置。
- 确认：按钮压印 180ms + `submitting`，不显示“投票成功”。
- 提交确认：服务端返回后，本人选择和提交进度更新。
- 揭票：`game.vote_resolved` 后，从真实票数增长到实际值。
- 平票/PK/弃权：先文字，再视觉区分，不只使用红/绿。

### 7.11 结局揭晓

- 首屏先结论，后因果。
- 时间线和证据链由服务端结果构建。
- AI 点评异步加载，不能阻塞规则结果。
- 如果用户在动画中刷新，直接打开结果页终态。
- 如果服务端修正结果，取消旧动画并从新结果构建，不能播放“旧赢家”。
- Reduced Motion 使用静态结果与可展开时间线。

### 7.12 环境动效

允许：

- 低密度尘、薄雾、窗外雨、火光微动。
- 隐藏 Tab 时暂停。
- 离开场景时卸载。
- 阅读区上方的雾/雨必须被遮罩。

禁止：

- 正文后方无限 Shimmer。
- 全屏粒子/循环扫光。
- 滚动劫持。
- 每张卡独立呼吸。
- 每秒重排的计时闪烁。

### 7.13 性能与清理

- 高频动画只使用 transform 与 opacity。
- 不逐帧更新 React State。
- 粒子、模糊和 Canvas 有上限与低性能 fallback。
- Motion Presence 项卸载时清理 layout group 与订阅。
- 页面切换取消未完成的临时序列；不得继续向新路由传输私密动画层。
- IntersectionObserver、媒体查询监听、键盘监听在组件卸载时移除。

Motion 可接受性：

- 快速连续触发不积压动画队列。
- 焦点在动画过程中可用。
- 动画中 Tab/Enter 不会触发隐藏操作。
- 动画失败不改变业务状态。

---

## 8. 状态与实时交互设计

### 8.1 统一状态模型

```ts
type SurfaceState =
  | "initial"
  | "loading"
  | "partial"
  | "ready"
  | "empty"
  | "error"
  | "disabled"
  | "permission_denied"
  | "unavailable"
  | "offline"
  | "reconnecting"
  | "stale"
  | "conflict"
  | "settling"
  | "success";
```

`success` 指状态已由服务端确认并持久呈现，不指 Toast 出现或动画结束。

### 8.2 通用状态表面

| 状态 | 视觉 | 文案 | 行为 |
| --- | --- | --- | --- |
| Loading | 与最终布局同形的骨架 | 正在加载 X | 保留导航和稳定主操作占位 |
| Empty | 安静留白 + 合法下一步 | 为什么为空 | 提供创建/清除筛选/返回 |
| Error | 局部错误面 + requestId | 什么发生、什么未发生 | 重试、返回或联系帮助 |
| Success | 持久状态变化 | 已确认的当前结果 | 可继续下一步 |
| Disabled | 稳定控件 + 原因 | 不可用原因 | Focus/Hover 可读，不反复抖动 |
| Offline | 全局网络条 + 局部只读 | 离线，显示最后确认内容 | 禁止写操作 |
| Reconnecting | 连接条 + 序号/状态 | 正在恢复 | 保留最后确认投影，阻止重复提交 |
| Stale | 版本提示 | 场景/状态已更新 | 刷新权威状态，保留输入 |
| Conflict | 明确冲突内容 | 哪个版本或动作冲突 | 让用户选择安全恢复 |
| Settling | 锁定操作 + 本地进度 | 等待服务端结算 | 不猜胜负 |

### 8.3 页面状态矩阵

| 页面 | Loading | Empty | Error | Success | Disabled | Offline | Reconnecting |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 首页 | 分区骨架 | 无历史/无推荐 | 单区失败不阻塞 | 推荐/继续可用 | 未开放模式 | 缓存公开内容 | 继续入口显示网络条 |
| 登录 | 会话恢复骨架 | 无表单数据 | 凭证/网络错误 | 登录后返回目标 | 协议未勾选 | 提交禁用 | 会话恢复失败可重试 |
| 大厅 | 旧结果 + 局部骨架 | 无筛选结果 | 搜索失败保留旧结果 | 筛选结果就绪 | 快速加入不可用 | 浏览只读 | 快速加入禁用并提示 |
| 剧本详情 | 标题/元信息优先 | 无评论/推荐 | 下架/无权 | CTA/权限就绪 | 创建条件不足 | 公开内容可读 | 收藏/分享重试 |
| 工坊 | 草稿/表单骨架 | 无草稿/未上传 | 文件/配额/扫描错误 | 可开始生成 | 权利/模式/配额禁用 | 上传暂停 | 上传任务状态未知 |
| 生成进度 | 任务骨架 | 任务不存在 | 节点/模型失败 | 候选可编辑 | 不可重试动作 | SSE 中断保留终态 | SSE 按事件 ID 重连 |
| 编辑器 | 对象树优先 | 空模板/空对象类 | 保存/校验冲突 | 已保存/可发布 | 阻断问题/只读 | 本地草稿可读 | 保存队列暂停 |
| 创建房间 | 内容摘要骨架 | 无可用内容 | 版本/人数错误 | 创建成功进入等待 | 人数/权限不足 | 创建禁用 | 创建校验重试 |
| 房间等待 | 席位优先 | 空席/无聊天 | 房间/开始失败 | 全员准备 | 准备/开始不合法 | 席位保留只读 | 重连并重建席位 |
| 剧本杀 | 阶段骨架 | 无线索/无聊天 | 搜证/投票/DM 失败 | 阶段与动作就绪 | 非当前阶段动作 | 最后确认场景只读 | 恢复快照后开放动作 |
| 狼人杀 | 席位/阶段骨架 | 无遗言/无投票权 | 行动/技能/结算错误 | 合法夜间/白天动作 | 非本人回合 | 保留公共状态 | 私密行动等待重同步 |
| RPG | 场景/地图骨架 | 无任务/无队友 | 解析/移动/NPC 失败 | 场景与合法动作就绪 | 资源/目标不可用 | 最后场景只读 | 重建场景版本后恢复 |
| NPC 对话 | 历史骨架 | 首次问候 | NPC 离场/安全替换 | 对话可发送 | 非在场/不可对话 | 历史只读 | 重连后最后正式消息 |
| 结算 | 规则结果优先 | 无统计仍显示结果 | 结果缺失/异常 | 结果和复盘入口 | 分享条件不足 | 已确认结果可读 | 结果不重播 |
| 复盘 | 时间线骨架 | 当前视角无事件 | 权限/版本错误 | 事件链可读 | 私聊/角色数据不可见 | 缓存事件只读 | 重新请求授权视角 |
| 个人中心 | 摘要优先 | 无简介/成就/作品 | 审核/保存失败 | 资料与隐私已保存 | 受限账号/字段禁用 | 公开资料可读 | 保存队列暂停 |
| 历史 | 列表骨架 | 无记录 | 权限失效/数据异常 | 记录可复盘 | 过期不可继续 | 可见摘要只读 | 继续房间校验重试 |
| 社区 | 信息流骨架 | 无内容/无评论 | 评论/举报失败 | 内容与互动就绪 | 未完成对局不可评分 | 公开内容可读 | 评论提交状态未知 |
| 后台 | 表格骨架 | 无匹配记录 | 查询/审批失败 | 审核动作已审计 | 无权限/需审批 | 只读缓存 | 审计结果重查 |

### 8.4 关键操作状态机

```text
idle
→ validating
→ submitting
→ accepted / rejected / stale / timeout / degraded
→ confirmed
→ idle or terminal
```

| 状态 | UI 表现 | 服务端关系 |
| --- | --- | --- |
| Validating | 本地合法性与目标检查 | 不改变状态 |
| Submitting | 稳定按钮 + “提交中” | 幂等键 + expected version |
| Accepted | 可显示“已提交，等待结算” | 命令已接收但结果未公开 |
| Rejected | 原因码 + 恢复动作 | 状态未改变 |
| Stale | 版本已变化 | 请求快照/刷新动作 |
| Timeout | 不确定是否接收 | 使用原幂等键重试 |
| Degraded | AI 或非关键能力降级 | 规则状态仍有效 |
| Confirmed | 持久 UI + 事件/结果 | 权威状态已更新 |

关键操作不显示乐观结果：

- 搜证获得线索
- 投票成功/票型
- 夜间行动结果
- 技能触发结果
- 淘汰/身份揭示
- 物品使用与背包变更
- 任务接受/放弃
- 世界移动与剧情选择
- 发布、下架、开局

可乐观并回滚：

- 收藏
- 已读标记
- 非关键 UI 偏好
- 本地草稿保存提示（最终仍以服务端保存为准）

### 8.5 实时连接状态

| `connectionState` | 全局表现 | 页面行为 |
| --- | --- | --- |
| `connecting` | 细进度条 | 显示快照骨架 |
| `connected` | 无横幅 | 等待 subscribe/snapshot |
| `syncing` | “正在同步至事件 X” | 禁止关键写操作 |
| `ready` | 无横幅 | 全部能力按投影启用 |
| `degraded` | 黄色状态条 | 暂停增量应用，请求 resync |
| `reconnecting` | 网络条 + 当前阶段保留 | 禁止重复提交，保留输入 |
| `offline` | 灰色状态条 | 最后确认内容只读 |
| `disconnected` | 错误/重试入口 | 关闭私密临时层 |

状态条不遮挡阶段、计时和固定主操作。恢复后短时显示“已同步到事件 #185 / 状态版本 18”，随后可收起。

### 8.6 事件去重、乱序与补偿

- 同一 `event_id` 只应用一次。
- 同一 `stream + stream_id` 小于当前 `seq` 的事件忽略。
- 未来序号形成缺口时停止应用，发送 `resync.request`。
- `room` 和 `game` 序号独立，不可交叉比较。
- `room_chat` 使用独立 `chat_seq`，不得与 room/game event seq 混用。
- 大快照优先通过 HTTP `G01`/`G03` 获取，不在单条 WS 消息阻塞。
- 重连清空旧实时投影后应用快照和 delta，不能把两份状态双向合并。
- 未确认 REST mutation 携带原幂等键重试，展示原结果。
- `IDEMPOTENT_REPLAY` 是成功回放，不提示用户重复提交。

### 8.7 计时与阶段推进

```ts
const remainingMs =
  new Date(phase.deadlineAt).getTime() -
  (Date.now() + serverTimeOffsetMs);
```

- `deadlineAt` 来自服务端；客户端只渲染剩余时间。
- 倒计时到 0 后显示“等待服务端推进”，主操作只按 `available_actions` 启用。
- `game.timer_updated` 更新暂停/恢复后的截止时间。
- 后台 Tab 恢复时重新获取权威投影，不只依赖本地计时。
- 动画不负责触发下一阶段、淘汰、投票关闭或结算。

### 8.8 多人冲突

| 冲突 | 示例 | UI 恢复 |
| --- | --- | --- |
| 最后席位 | 两玩家同时加入 | 失败方看到“房间已满”，回大厅保留筛选 |
| 独占线索 | 两玩家同时搜证 | 失败方看到“已被发现”，不扣次数 |
| 版本冲突 | 投票/行动基于旧版本 | 刷新投影，保留选择前确认 |
| 编辑冲突 | 两个协作者保存同一对象 | 重新加载 / 保留本地差异 |
| 资源争抢 | RPG 物品/路线被队友先使用 | 显示当前状态与替代动作 |
| 房主迁移 | 房主掉线 | 新房主广播，旧房主重连成普通玩家 |
| 阶段关闭 | 用户提交时阶段变化 | 拒绝 + 新阶段合法动作 |

### 8.9 AI 降级状态

| 事件 | UI | 是否阻塞规则 |
| --- | --- | --- |
| `ai.run.degraded` | “已切换为模板主持/回应” | 否 |
| AI 首 Token 慢 | 局部“正在叙述” | 否 |
| AI 安全拦截 | 安全模板文本 + 简短说明 | 否 |
| NPC 离线 | 结构化动作仍可执行 | 否 |
| 复盘 AI 失败 | 隐藏 AI 点评，保留规则时间线 | 否 |

AI 状态不能覆盖当前搜证、投票、技能或世界动作。任何内部供应商错误、Prompt 或推理链不进入玩家 UI。

### 8.10 离线与恢复边界

- 仅 D0/D1 公开数据可进入持久缓存。
- D2-D4 私密投影只保留在当前运行内存。
- 已打开页面的最后确认私密投影可在离线时继续阅读，但不得写入持久缓存或分享。
- 刷新后若离线，无法重新获得私密投影，应显示登录/连接恢复状态，而不是旧秘密。
- 退出登录、房间结束或账号受限时清理游戏 Store。

### 8.11 持久反馈与 Toast

Toast 仅用于：

- 复制邀请/链接。
- 保存草稿（辅助）。
- 收藏/已读。
- 非关键设置已更新。

必须持久反馈：

- 线索获得。
- 投票已确认。
- 夜间行动已提交/结算。
- 技能结果。
- 出局/身份揭示。
- 世界状态变化。
- 发布/下架。
- 结算结果。

---

## 9. API 数据映射

### 9.1 数据边界

```text
OpenAPI / JSON Schema generated types
  → API adapter / WebSocket reducer
  → view model at feature boundary
  → React component
```

- 前端不直接消费数据库 Entity。
- REST 只消费 Public/Viewer/Private/Runtime 投影 DTO。
- WebSocket `payload` 已经是当前 Viewer 的权限投影。
- 不把 `secret_payload_id`、密文、完整身份表、Prompt 或内部推理接入 UI。
- 生成类型禁止手改；view model 只做命名和展示归一，不改变权限语义。

### 9.2 路由数据映射

| Route | REST / SSE | 实时 | 客户端状态 | 缓存/失效 |
| --- | --- | --- | --- | --- |
| `/home` | `U04`、`S02`、`R02`；推荐/精选聚合待补 | 无 | `userSummaryQuery`、`recommendedQuery`、`roomListQuery` | 60s；登录/回到页面刷新 |
| `/auth` | `U02`、`U01`、`U03`、`R08` | 无 | 表单 local state | 不缓存凭证；成功清理旧会话 |
| `/lobby` | `S02`、`R02`、`R04` | 无 | `lobbyStore` 仅视图；筛选在 URL | 筛选结果 30-60s；加入后失效 room |
| `/scripts/:id` | `S03`、`C01`、`C03`、`C05`、`C06` | 无 | `scriptDetailQuery` | 5min；收藏后局部失效 |
| `/workshop` | `U04`、`S01`、`S05`、`S06`、`S07`；草稿列表待补 | 无 | `workshopStore` 表单/步骤 | 表单不缓存为公共；提交后失效 drafts |
| `/workshop/:jobId/progress` | `S08`、SSE `S09`、`S10` | SSE `generation.*` | `jobQuery` + 本地 SSE 投影 | `staleTime: 0`；SSE 断线按 `last_event_id` |
| `/workshop/:draftId/edit` | `S13`、`S14`、`S15`、`S16`、`S17`、`S20`、`S11` | 无 | `editorStore` 草稿/选中/校验 | 编辑内容不 stale；保存以 row_version |
| `/rooms/create` | `S03`、`R01` | 无 | 表单 local/Store | 提交后失效 room/continue |
| `/rooms/:roomId/waiting` | `R03`、`R06`、`R07`、`R09`、`R10`、R13/R14 | WS `room.*`、`room_chat.*` | `roomStore` 投影 | `staleTime: 0`；进入先快照 |
| `/play/murder/:roomId` | `G01`、`M01-M08`、`G04-G08` | WS `game.*`、`ai_stream` | `gameStore` | 首屏快照不 stale；后续只 WS |
| `/play/werewolf/:roomId` | `G01`、`W01-W07`、`G04-G08` | WS `game.*`、`ai_stream` | `gameStore` | 同上 |
| `/play/rpg/:roomId` | `G01`、`P01-P11`、`G04-G08` | WS `game.*`、`ai_stream` | `gameStore` | 同上 |
| `/play/:roomId/npc/:npcId` | `P06`、`P07`、`P01`、`P05` | WS `npc.dialogue_generated`、`ai_stream` | `gameStore` + local draft | 历史分页；正式消息持久 |
| `/play/:roomId/result` | `G06`、`A01` | WS `game.result_ready` | `resultQuery` + latest result | 结果不可变；AI 状态单独失效 |
| `/play/:roomId/replay` | `G07`、`G02`、`A01` | 无 | `replayStore` 视角/选中事件 | 5min；切换 viewer scope 重新请求 |
| `/me` | `U04`、`U05`、`U06`、`U07-U10` | 无 | `meQuery`、local edit state | 60s；资料保存后失效 |
| `/me/history` | 完整历史接口待补；`U04.include=recent_games` 只适合近期 | 无 | filter URL + historyStore | 待接口定义 |
| `/community` | `S02`、`C01-C07` | 无 | feedStore/filter URL | 分页缓存 60s；评论后局部失效 |
| `/admin/*` | `docs/api/admin-api.md` 基线 | 无玩家 WS | adminStore | 独立敏感缓存策略 |

### 9.3 Query 与 Zustand 分工

| 数据类型 | TanStack Query | Zustand | 不得进入 Zustand 的字段 |
| --- | --- | --- | --- |
| 剧本详情 | 是 | 仅选中对象/Tab | 分享 token 原文 |
| 编辑器 | 对象请求 + mutation | 当前对象、草稿 diff、UI 状态 | 其他角色秘密 |
| 房间快照 | 首屏 Query | 实时席位投影 | 密码明文 |
| 游戏投影 | 首屏 Query | 实时投影与连接 | 权威身份表/秘钥 |
| 生成任务 | 任务快照 Query | 当前步骤/问题选择 | Prompt、原文切片 |
| 结算/复盘 | Query | 当前视角/选中事件 | 未授权私聊 |

### 9.4 WebSocket 事件到 UI

| Event | UI 更新 | 不做什么 |
| --- | --- | --- |
| `server.ready` | 显示连接与协议版本 | 不显示房间数据 |
| `room.snapshot` | 重建席位、准备、房主 | 不合并旧席位 |
| `room.player_joined/left/ready` | 更新席位与 readiness | 不推断角色 |
| `room.host_transferred` | 房主标识迁移 | 不改变游戏身份 |
| `room.message_received` | 等待房聊天消息 | 不写入游戏事件或 state version |
| `game.snapshot` | 重建阶段、公共/私密投影 | 不清除当前已确认结果之外的本地草稿 |
| `game.phase_changed` | 阶段条、主区、合法动作 | 不提前启用下一阶段动作 |
| `game.state_updated` | 应用 viewer 状态补丁 | 不把其他玩家私密补丁写入 |
| `game.clue_unlocked` | 档案与揭示序列 | 不在事件前播放 |
| `game.message_received` | 对应频道消息 | 不加入其他频道 |
| `game.vote_submitted` | 提交进度 | 不展示票型 |
| `game.vote_resolved` | 真实票型与后果 | 不重算规则 |
| `game.player_eliminated` | 公共状态/遗言入口 | 不推断角色 |
| `game.skill_triggered` | 授权技能结果 | 不显示其他阵营结果 |
| `game.timer_updated` | Deadline / 暂停 | 不自行切阶段 |
| `game.result_ready` | 结果入口/结果页 | 不等待 AI 点评 |
| `game.incident_frozen` | 冻结状态与支持入口 | 不清空最后确认结果 |
| `npc.dialogue_generated` | 正式 NPC 消息 | 不把流式片段写成事实 |
| `ai.run.degraded` | 降级提示 | 不阻塞规则操作 |
| `ai.stream.chunk` | 安全展示片段 | 不改变 `state_version` |
| `server.resync_required` | 请求快照 | 不继续应用增量 |

### 9.5 SSE 生成任务事件到 UI

| SSE Event | UI |
| --- | --- |
| `generation.progress` | 更新总 `progress_bp`、当前步骤 |
| `generation.step_changed` | 更新节点状态、并行分支 |
| `generation.failed` | 显示错误、重试/人工确认入口 |
| `generation.succeeded` | 显示候选产物与进入编辑器 |

内部 `ai.node.*` 事件需由 Go 映射为当前 API 契约。前端不直接依赖未公开内部事件名。

### 9.6 Mutation 规范

```ts
type CriticalMutationMeta = {
  idempotencyKey: string;
  expectedVersion: number;
  clientRequestId?: string;
  permissionContext: string;
};
```

- 生成 `Idempotency-Key`，同一意图重试复用。
- `expected_version` 必须来自最新服务端投影。
- 成功后使对应 Query 失效，并以 WS 事件或返回 DTO 更新投影。
- 失败按稳定错误码分支，不解析自然语言 message。
- 关键操作在 pending 时禁用重复点击，但保持控件尺寸。

### 9.7 错误码到 UI

| Error code | UI |
| --- | --- |
| `AUTH_REQUIRED` | 登录并在白名单 returnTo 后返回 |
| `FORBIDDEN_VIEW` | 隐藏敏感内容，显示安全返回 |
| `STATE_VERSION_CONFLICT` | “状态已更新”，刷新投影 |
| `meta.idempotent_replay=true` | 展示首次结果，不重复执行；不按错误 Toast 处理 |
| `PHASE_NOT_ALLOWED` | 当前阶段不可用原因 + 刷新 |
| `SEARCH_QUOTA_EXHAUSTED` | 显示剩余次数，进入合法下一步 |
| `CLUE_ALREADY_CLAIMED` | 点位状态更新，不扣次数 |
| `TARGET_INVALID` | 清除非法目标，更新合法候选 |
| `VOTE_CLOSED` | 锁定面板，刷新票型 |
| `AI_TIMEOUT` | 模板降级，不阻塞规则输入 |
| `AI_SAFETY_BLOCKED` | 替换输出，明确内容不可用 |
| `QUOTA_EXCEEDED` | 显示配额与恢复建议 |
| `REALTIME_DISCONNECTED` 客户端连接态 | 重连条，保留最后确认状态；它不是服务端错误码 |
| `REPLAY_NOT_READY` | 结构化结果先展示，AI 点评延迟 |
| `SERVICE_UNAVAILABLE` | 停止写操作，显示维护信息 |

### 9.8 契约缺口

以下页面/数据在 PRD 中存在，但当前 `API_DESIGN.md` 未提供明确公共接口。实现前必须补契约，不能由 UI 伪造：

| 需求 | 当前情况 | UI 处理 |
| --- | --- | --- |
| 首页推荐、官方精选、近期聚合 | 只有 `S02/U04/R02` 基础数据 | 使用公开热门兜底，不显示假推荐 |
| 等待房聊天 | ADR-006 已冻结 `room_messages`、R13/R14 与 `room_chat` 流 | 按正式契约实现；聊天故障不阻塞开始/准备 |
| 草稿/生成任务项目列表 | `S03/S11` 可部分推断，无聚合列表 | 工坊显示当前项目/空状态 |
| 完整游戏历史筛选 | `U04.include=recent_games` 不足 | 历史页等待 `GET /me/games` 类接口 |
| 成就列表 | PRD 有，API 无 | 不渲染假徽章，显示能力待接入 |
| 我的剧本聚合 | 需作者/项目列表接口 | 使用 S03/S11 兜底，标注未完成 |
| 通知中心 | 导航需要通知 | 只显示入口状态，未接接口不伪造未读 |
| 全站搜索 | PRD 搜索，API 只按剧本搜索 | 搜索仅限剧本/房间，不假装全站检索 |
| 运营后台基础 API | ADR-007 已冻结为公开 MVP 门禁；详细契约见 `docs/api/admin-api.md` | 内测可人工审核；公开 MVP 必须接入最小后台 |
| 管理员敏感字段查看申请 | API 无 | UI 显示审批占位，不展示正文 |

### 9.9 组件数据来源速查

| 组件 | 主要 REST | 实时 | Store |
| --- | --- | --- | --- |
| GameRoom | `G01`、`R03` | `room.*`、`game.*` | `gameStore` |
| PlayerCard | `G01` | `game.player_eliminated`、`game.vote_submitted` | `gameStore.players` |
| CharacterCard | `M01/W01` | `game.state_updated` | `gameStore.privateState` |
| ClueCard | `M05/M04` | `game.clue_unlocked` | `evidenceStore` |
| EvidencePanel | `M05` | `game.clue_unlocked` | `evidenceStore` |
| RelationshipGraph | `S13/S20/M01` | `game.state_updated` | `editorStore`/`gameStore` |
| Timeline | `S13/G01/G02/G07` | `game.state_updated` | `timelineStore` |
| NPCDialogue | `P06/P07` | `npc.dialogue_generated`、`ai.stream.chunk` | `npcStore` |
| StoryScene | `G01/P01/M03` | `game.phase_changed`、`game.state_updated` | `gameStore.scene` |
| GamePhaseIndicator | `G01` | `game.phase_changed`、`game.timer_updated` | `gameStore.phase` |
| VotingPanel | `M07/M08/W05` | `game.vote_submitted`、`game.vote_resolved` | `gameStore.vote` |
| MapExplorer | `P01/P02/P03` | `game.state_updated` | `gameStore.map` |
| QuestPanel | `P08/P09` | `game.state_updated` | `gameStore.tasks` |
| GenerationWorkflowPanel | `S08/S10` | SSE `generation.*` | `jobStore` |

---

## 10. 响应式规范

### 10.1 断点

| Token | 范围 | 设计目标 |
| --- | --- | --- |
| `compact` | `< 640px` | 单列、固定主操作、Tab/Sheet |
| `medium` | `640-1023px` | 双列或场景优先，侧栏 Drawer |
| `wide` | `1024-1439px` | 双栏/三栏工作区，完整操作 |
| `ultrawide` | `>= 1440px` | 最大内容宽度、辅助面板扩展 |

不使用设备名决定布局；根据「主操作是否仍需可见」「是否需要并排比较」「是否有键盘输入」决定断点行为。

### 10.2 响应式原则

- 移动端是重新编排，不是压缩桌面三栏。
- 当前阶段、计时和主操作在 360px 宽度下必须可见。
- 不使用固定高度截断中文标题或按钮。
- 主要触控目标最小 44x44px；相邻目标至少 8px 间距。
- 抽屉、Tab、Sheet 必须有可见标题与关闭方式。
- 底部固定操作使用：

```css
padding-bottom: max(16px, env(safe-area-inset-bottom));
```

- 横向滚动仅用于明确的证据板/关系图等空间工具，必须提供列表替代和滚动提示。

### 10.3 页面响应式矩阵

| 页面 | Compact | Medium | Wide | Ultrawide |
| --- | --- | --- | --- | --- |
| 首页 | 继续 → 模式 → 推荐 | 两列模式/推荐 | 完整模式 rail + 内容 | 增加社区/最近辅助栏 |
| 大厅 | 列表 + 筛选 Drawer | 2 列结果 | 左筛选 + 网格 | 3-4 列结果 |
| 剧本详情 | 封面上 + 固定 CTA | 封面/元信息上下 | 内容 + sticky CTA | 内容 + 相似剧本侧栏 |
| 工坊 | 单列步骤 | 表单 + 摘要上下 | 步骤 + 表单 + 摘要 | 表单更宽，摘要固定 |
| 生成进度 | 纵向节点 | 时间线 + 问题 Sheet | 时间线 + 问题侧栏 | 时间线 + 产物预览 |
| 编辑器 | 查看/轻编辑 | 对象树 + 表单 | 四区 | 增加原文并排宽度 |
| 创建房间 | 摘要上 + 表单 | 双列摘要/表单 | 左摘要 + 右表单 | 增加规则模板预览 |
| 房间等待 | 席位/聊天 Tab | 席位 + 聊天 Drawer | 席位 + 右侧栏 | 大席位网格 + 规则 |
| 剧本杀 | 剧情/线索/聊天/角色 Tab | 场景 + Drawer | 主区 + 右栏 | 主区更宽，证据板可并列 |
| 狼人杀 | 紧凑席位 + 底部 Sheet | 席位环 + 侧栏 Drawer | 席位环 + 侧栏 | 更大席位环 |
| RPG | 场景 + Drawer | 场景 + 地图 Sheet | 三区 | 左侧地图/右侧任务扩展 |
| NPC | 全屏对话 | 大 Sheet | 侧面板/大面板 | 场景 + 对话 + 队伍三区 |
| 结算 | 结果/统计纵向 | 结果 + 统计 | 结果 + 因果栏 | 增加模式可视化区域 |
| 复盘 | 顶部阶段 + 折叠事件 | 时间线 + 详情 | 三区 | 增加对照/引用事件栏 |
| 个人中心 | 单列 Tab | 摘要 + Tab | 摘要 + 内容 | 摘要 + 分组侧栏 |
| 历史 | 单列卡片 | 两列 | 列表 + 侧筛选 | 列表 + 更宽摘要 |
| 社区 | 单列流 | 两列 | 内容流 + 侧栏 | 内容流 + 专题/作者栏 |
| 后台 | 表格横向受控滚动 | 表格 + 详情 Drawer | 列表 + 详情 | 列表 + 详情 + 审计栏 |

### 10.4 游戏页响应式铁律

- 1024px 以下不保留三列窄面板；转为场景主区 + Drawer/Sheet。
- 移动端固定主操作不进入滚动消失区。
- 证据/Search Point 有列表入口，不能依赖场景热点。
- 关系图/地图有可键盘访问的列表或邻接视图。
- 场景移动裁剪必须保留主体；不用简单 `object-position: center` 导致 NPC 或证据被裁掉。
- 聊天/输入框出现软键盘时，滚动容器与固定操作不重叠。
- 私密内容 Sheet 关闭后不保留在页面可见的背景层。

### 10.5 断点字号切换

```css
:root {
  --mv-text-display-size: 56px;
  --mv-text-display-leading: 64px;
  --mv-text-h1-size: 40px;
  --mv-text-h1-leading: 48px;
}

@media (max-width: 639px) {
  :root {
    --mv-text-display-size: 38px;
    --mv-text-display-leading: 46px;
    --mv-text-h1-size: 32px;
    --mv-text-h1-leading: 40px;
  }
}
```

这些值不随视口宽度连续缩放，避免字号不可控。

### 10.6 验收视口

最低检查：

- 360x800
- 390x844
- 768x1024
- 1024x768
- 1440x900
- 1920x1080

检查内容：

- 文本不裁剪、不重叠。
- 主操作不离开视口。
- 场景主体仍可辨认。
- Tab/Sheet 可打开、关闭、返回焦点。
- 无意外水平滚动。
- iOS Safari 与 Android Chrome 软键盘状态可用。

---

## 11. 无障碍规范

### 11.1 基线

- 目标：WCAG 2.2 AA 为主，关键游戏流程以可完整键盘操作为门禁。
- 正文文本对比 ≥ 4.5:1；大字 ≥ 3:1；UI 边界与 Focus ≥ 3:1。
- 信息不能只依赖颜色、声音、动画或位置。
- 200% 文本缩放和浏览器缩放时关键操作可见。
- 所有交互控件有可见 Focus。

### 11.2 键盘与焦点

| 流程 | 要求 |
| --- | --- |
| 首页/大厅 | Tab、Enter、Space、筛选控件全键盘可用 |
| Auth | 字段顺序、错误聚焦、验证码重发可用 |
| 工坊 | 上传按钮、步骤切换、提交错误可键盘定位 |
| 生成进度 | 节点可读，重试/取消可聚焦 |
| 编辑器 | 对象树可用方向键/Enter；表单标签关联 |
| 房间 | 席位、准备、邀请、房主动作可键盘操作 |
| 游戏 | 当前主操作在文档顺序中优先；搜索点/Tab/Drawer 可键盘 |
| 证据板/关系图/地图 | 提供列表或邻接表等价操作 |
| 结算/复盘 | 时间线可逐事件浏览，AI 说明不被用作按钮 |

规则：

- 页面转场后焦点落到主标题或首个有意义操作。
- Dialog/Sheet 打开时焦点进入，关闭返回触发控件。
- 非模态侧栏不劫持焦点。
- 动画过程中已可交互元素保持合理焦点顺序。
- 不允许 `div onClick` 代替按钮。

### 11.3 语义与 ARIA

- 页面：`header`、`nav`、`main`、`aside`、`footer` 语义结构。
- 阶段：`aria-live="polite"`，只在阶段变化时宣告一次。
- 计时：不每秒 `aria-live`；至少每分钟或关键阈值宣告。
- 线索揭示：Dialog/Sheet 有 title，读屏先读标题、范围、来源和主体。
- 投票：候选用 radio group 或 list + explicit selection。
- 角色/线索计数：数值变化通过文本状态，不只依赖动画。
- AI 流式文本：不逐 Token 播报；完成安全终稿后宣告完整段落。
- 图标按钮：`aria-label` + Tooltip；装饰图标 `aria-hidden`。
- 私密范围：图标必须有文字标签，读屏可理解。

### 11.4 空间界面替代

RelationshipGraph、Evidence Board、MapExplorer 必须提供：

- 可切换的列表/邻接表/路线列表。
- 节点名称、状态、可见性、可达条件和操作。
- 与空间选择等价的键盘焦点与选择行为。
- 关系/路线变化用文本说明。

Canvas/SVG 不能是唯一交互载体。

### 11.5 动效与感官

- `prefers-reduced-motion` 提供静态/短过渡路径。
- 震动、闪屏、强烈缩放、突发大音量默认避免。
- 提供：
  - 减少动效
  - 环境声量
  - 音效声量
  - 屏幕震动/闪烁开关
  - 触觉反馈开关（若未来支持）
- 设置即时生效，不要求用户重播剧情。
- 动画跳过按钮在揭示开始后可达。

### 11.6 声音

- MVP 无语音也需要环境/提示音设置。
- 声音不是唯一确认。
- 重要声音有文本等价说明。
- 自动播放遵循浏览器策略；首次交互后才可开始环境音。
- 隐藏 Tab 或离开页面暂停声音。

### 11.7 内容与认知

- 每个错误说明：发生了什么、什么没有发生、下一步。
- 禁用动作提供原因，不用红色仅表示。
- 长剧情按段落和标题组织；不只给一整屏连续文本。
- 计时压力有非紧迫提示选项。
- 恐怖/闪烁/敏感主题提前显示内容提示。

### 11.8 无障碍测试矩阵

| 测试 | 方式 |
| --- | --- |
| 主流程键盘完成 | 关闭鼠标，逐页执行 |
| Focus 可见 | 暗色、亮色、场景背景、Overlay |
| 文本缩放 | 200% 与 400% 浏览器缩放 |
| Reduced Motion | OS 设置 + 应用设置 |
| 读屏摘要 | 阶段、错误、线索、投票、结果 |
| 地图/关系图 | 列表替代完成关键任务 |
| 颜色缺失 | Grayscale/对比检查 |
| 触控 | 44px 目标与 Sheet 关闭 |
| 软键盘 | iOS/Android 输入与底部操作 |

---

## 12. 图片与素材需求

### 12.1 素材原则

- 需要被检查的地点、人物、物件、证据使用真实内容图或专门生成的位图，不使用模糊情绪图。
- 图片不承载 UI 文字；标题、来源、可见范围和动作由 DOM 渲染。
- 背景图必须保留 UI 可读区，必要时提供局部纹理/暗色 Mask。
- 角色立绘与证据图有稳定主体裁剪；移动端使用专门裁切，不只中心裁剪。
- 纹理、纸张、金属印章是材料，不是覆盖全屏的装饰。
- 禁用虚构案件截图、通用侦探剪影、紫色霓虹、玻璃拟物、抽象光球和 Emoji 素材。

### 12.2 素材清单

| Asset ID | 内容 | 模式 | 尺寸/格式 | 优先级 | Fallback |
| --- | --- | --- | --- | --- | --- |
| `img-home-murder` | 案卷/调查空间模式主视觉 | 剧本杀 | 2560x1440、1080x1440、AVIF/WebP | P0 | 墨色 + 纸面 + 模式名 |
| `img-home-werewolf` | 仪式席位/夜间村落 | 狼人杀 | 2560x1440、1080x1440 | P0 | 月银色块 + 席位几何剪影 |
| `img-home-rpg` | 可探索世界远景 | RPG | 2560x1440、1080x1440 | P0 | 苔绿/铜色渐变 + 地图线 |
| `img-werewolf-night` | 夜晚村落/仪式广场背景 | 狼人杀 | 2400x1350、1080x1350 | P0 | 冷灰夜plate |
| `img-werewolf-dawn` | 同一地点的黎明plate | 狼人杀 | 2400x1350、1080x1350 | P0 | 暖灰黎明plate |
| `img-rpg-region-main` | RPG 主世界区域 | RPG | 2560x1440、1080x1440 | P0 | 地形色块 + 路线层 |
| `img-rpg-region-alt` | RPG 区域变体 | RPG | 2560x1440、1080x1440 | P1 | 主区域 + 状态 tint |
| `img-event-case-open` | 案件开场场景 | 全模式 | 2400x1350、1080x1350 | P0 | 场景标题 + 材料色块 |
| `img-event-scene-enter` | 门槛/场景切换背景 | 全模式 | 2400x1350、1080x1350 | P1 | 前一场景静态叠化 |
| `img-event-npc-enter` | NPC 所在环境背景 | RPG/剧本杀 | 2400x1350、1080x1350 | P1 | 场景背景 + 立绘占位 |
| `img-event-clue` | 证据展示背景/材料 | 剧本杀 | 2000x1250、1080x1350 | P0 | 纸面 + 标题 |
| `img-event-secret` | 封印卷宗/秘密卡片背景 | 全模式 | 2000x1250、1080x1350 | P1 | 纸面 + 封印图形 |
| `img-event-death` | 死亡/淘汰场景背景 | 狼人杀/RPG | 2400x1350、1080x1350 | P1 | 席位状态 + 低饱和背景 |
| `img-event-vote` | 投票/仪式桌面背景 | 剧本杀/狼人杀 | 2400x1350、1080x1350 | P1 | 席位/角色摘要 |
| `img-event-truth` | 真相重构场景 | 剧本杀 | 2400x1350、1080x1350 | P0 | 结构化时间线终态 |
| `img-event-ending` | 隐藏结局世界变体 | RPG | 2400x1350、1080x1350 | P1 | 结局标题 + 状态变量 |
| `img-npc-[id]` | NPC 半身/全身立绘 | RPG/剧本杀 | 1024x1536、透明 WebP/AVIF | P1 | 首字母 + 身份色块 |
| `img-evidence-[id]` | 线索物件/文档图 | 剧本杀/RPG | 1600x1200、800x1000 | P1 | 证据正文 + 来源 |
| `img-cover-[scriptId]` | 剧本封面 | 全模式 | 1200x1600、16:9 裁剪 | P1 | 模式化封面 |
| `tex-paper` | 纸张纹理/折痕 | 全模式 | 2048x2048，无缝或局部 | P1 | 纯纸色 |
| `tex-metal` | 金属/印章材料 | 全模式 | 1024x1024，透明 | P2 | CSS 边框/色块 |
| `tex-thread` | 证据关系线材料 | 剧本杀 | SVG/CSS | P2 | SVG 线 |
| `audio-room-ambience` | 房间底噪 | 全模式 | 60-120s 循环 OGG/MP3 | P2 | 无声 |
| `audio-transition` | 阶段提示音 | 全模式 | 1-3s OGG/MP3 | P2 | 仅视觉状态 |

建议资源规模：

- 3 个模式主页主视觉。
- 2 张狼人杀昼夜公共 plate。
- 9 个关键事件主背景及必要状态变体。
- 每人 1 张稳定 NPC 立绘 + 1 张移动裁切。
- 动态剧本封面、证据图由内容系统提供，不预生成假数据。
- 纸张/金属/线材纹理各 1-2 套，不按页面重复造。

### 12.3 通用 Prompt 模板

```text
Cinematic game environment art for MysteryVerse, [MODE], [LOCATION], [TIME/WEATHER],
layered background/midground/foreground composition, one clear focal subject,
material contrast between [PAPER/METAL/STONE/WOOD/ORGANIC MATERIAL],
coherent directional light from [LIGHT SOURCE], restrained contrast,
leave intentional negative space for UI overlays,
must remain readable with text and buttons on top,
no UI, no captions, no letters, no logos, no watermarks,
no generic SaaS dashboard, no purple-blue gradient, no neon, no glassmorphism,
no decorative orbs, no bokeh blobs, no excessive particles,
no stock-photo atmosphere, no fake detective props unrelated to the scene,
16:9 desktop composition with a deliberate 4:5 mobile crop.
```

Negative Prompt：

```text
UI panels, typography, logos, watermarks, purple neon, glass cards, glowing orbs,
bokeh background, generic office, SaaS dashboard, cartoon emoji, oversaturated colors,
illegible text, low-detail evidence, extreme blur, cluttered focal area,
unrelated detective clichés, fake newspaper headlines, visible gore without narrative reason.
```

风格约束：

- 不使用真实艺术家姓名。
- 不生成品牌 Logo、演员肖像或未授权 IP。
- 所有文字、人物名和证据事实由结构化内容决定，不直接写进图片。

### 12.4 模式主视觉 Prompt

#### `IMG-HOME-MURDER`

```text
A rain-darkened harbor manor study transformed into an active investigation desk.
Old paper dossiers, tide charts, a brass clock, ink stamps and red thread sit on a dark wood table.
One warm desk lamp isolates the evidence area; cold rain light enters from a tall window.
Focal point is a stopped clock and one evidence folder, with a readable empty zone above it.
Archive and investigation atmosphere, restrained cinematic realism, layered foreground rain on glass.
16:9, no people, no readable text, no UI, no logo, no purple gradient.
```

#### `IMG-HOME-WEREWOLF`

```text
A ritual village square seen from a raised oblique angle at night, nine empty seat markers arranged in a ring.
Moon-silver rim light, distant forest and low mist, one pale public path leading into the circle.
Dark oxblood accent appears only as a small hidden cloth edge, never as the dominant palette.
Focal point is the seat ring; the center stays clear for phase and vote overlays.
Quiet social tension, cinematic realism, 16:9, no people, no readable text, no UI.
```

#### `IMG-HOME-RPG`

```text
An explorable ancient trade road through a moss-covered mineral valley, copper route markers embedded in stone.
A distant gate and a lantern-lit waypoint form the focal hierarchy; weather is light fog after rain.
Moss green, wet slate, copper and pale daylight create depth without a one-note palette.
Foreground foliage frames the view but leaves a clean central reading zone.
Immersive living world, cinematic game environment, 16:9, no characters, no text, no UI, no logo.
```

### 12.5 关键事件 Prompt

#### `IMG-EVT-01` 案件开始

```text
Cinematic establishing shot of [CASE_LOCATION] at [TIME] in [WEATHER].
A single anomalous object in the lower-middle third suggests something is wrong:
[STOPPED_CLOCK / OVERTURNED_CHAIR / WET_FOOTPRINT / OPEN_DOOR].
The room architecture remains plausible and fully readable; warm practical light isolates the anomaly,
cool ambient light explains the shadows. Leave clear UI safe zones at top and bottom.
No victim body, no readable text, no private symbols, no UI, 16:9.
```

#### `IMG-EVT-02` 进入新场景

```text
A threshold transition between [PREVIOUS_LOCATION] and [NEW_LOCATION], preserving spatial direction.
Door frame, corridor edge or environmental occlusion creates a natural reveal into the new place.
The new focal subject is [FOCAL_OBJECT_OR_ROUTE]; old light remains as a narrow edge on the left.
No hidden exits or characters not approved by the scene state.
No text, no UI, no logos, 16:9 with mobile crop retaining the threshold and focal subject.
```

#### `IMG-EVT-03` NPC 登场

```text
A specific NPC [NPC_NAME], [VISUAL_IDENTITY], appears at [SCENE_LOCATION] in [WEATHER].
The character stands in the right third, face and silhouette readable, clothing and props consistent
with the world. The environment remains visible on the left for player context.
Lighting direction matches a practical source; no hidden faction color leaks.
No UI, no labels, no text; 16:9 and a 4:5 portrait-safe crop.
```

#### `IMG-EVT-04` 发现线索

```text
Macro cinematic still life of [EVIDENCE_OBJECT] discovered at [SOURCE_LOCATION].
Place it on a worn paper or stone surface with a credible source context in the blurred midground.
Evidence details must be inspectable: texture, edge, timestamp shape, stains, folds.
Warm evidence light isolates the object against a cool environment; leave a clean paper-like region
for title and fact overlays. No readable generated text, no UI, no decorative magnifying glass.
```

#### `IMG-EVT-05` 解锁秘密

```text
A sealed dossier or folded letter on a dark archive desk, wax seal and metal clasp opening.
The document is private evidence, not public spectacle. Warm narrow light on the seal,
soft shadow beyond the document edge. Paper fibers and fold marks are visible.
No character portrait leaks, no readable hidden text, no UI; 4:5 centered and 16:9 safe crop.
```

#### `IMG-EVT-06` 玩家死亡

```text
A public game seat or player position at [ROOM_LOCATION] after an authoritative elimination.
The chair/seat remains identifiable with nameplate area left empty for UI text.
Moonlight or scene light falls away from the seat; no gore, no generic skull overlay,
no hidden-role color, no body. Focal point is the empty state of the seat itself.
No text, no UI, no logo, 16:9.
```

#### `IMG-EVT-07` 投票结果

```text
An overhead ritual table with counting stones or vote tokens arranged around [CANDIDATE_SLOTS].
The exact number of tokens is left for DOM rendering; image provides material and spatial context.
Metal edges, paper ballots and moon-silver or amber highlights indicate collective decision.
No readable names, no fake tally numbers, no UI, no celebration particles, 16:9.
```

#### `IMG-EVT-08` 真相揭晓

```text
A cinematic reconstruction table showing [KEY_SCENE_OBJECT], several evidence fragments and a clear
empty area for a structured timeline overlay. Red thread or thin metal lines connect only plausible
evidence paths; blank spaces remain for event text. Warm paper and metal accents against subdued
environment image. Truth is serious and procedural, not fireworks. No text, no UI, no spoilers in art.
```

#### `IMG-EVT-09` 隐藏结局

```text
The same RPG world location [WORLD_LOCATION] after an irreversible hidden ending:
[FLOODED_GATE / OPENED_RELIC / CITY_ALARM / BROKEN_ALLIANCE].
The environment clearly shows the consequence through physical state change, not text.
One dominant world object and a stable result overlay area; copper and moss remain but danger color
comes from semantic UI. No fireworks, no text, no UI, no logo, 16:9 and mobile crop.
```

### 12.6 字体、图标与音频资源

| 资源 | 方案 | 优先级 |
| --- | --- | --- |
| `Noto Serif SC` | 自托管 WOFF2，用于叙事/标题 | P0 |
| `Noto Sans SC` | 自托管 WOFF2，用于 UI/正文 | P0 |
| `IBM Plex Mono` | 计时/序号/代码样式 | P0 |
| Lucide React | 唯一 UI 图标来源 | P0 |
| 环境音/阶段提示 | 可关闭、短小、循环有上限 | P2 |
| 纸张/金属纹理 | 低对比、可重复、提供纯色 fallback | P1 |

字体加载失败时使用系统字体栈，不能出现空白标题或布局跳变。图标不打包成图片精灵，使用可访问 SVG 组件。

### 12.7 资源加载与 fallback

- 图片使用 `srcset`/`sizes`、AVIF/WebP、稳定宽高比与 dominant color。
- 首屏关键场景图优先；评论/推荐缩略图懒加载。
- 背景图失败时显示材料化色块、地点名和场景摘要。
- 移动端单独裁剪，不用低清桌面图放大。
- 低性能 Tier：
  - A：图层 + 低密度环境效果 + 声音。
  - B：静态场景 + CSS 光斑 + 更短转场。
  - C：单张静态图 + 透明度切换，无粒子/位移/模糊。
- 隐藏 Tab 时暂停持续动效与音频。

---

## 13. 页面验收标准

### 13.1 全局验收门禁

以下任一项不满足，不得标记视觉或前端页面完成：

| Gate | 通过条件 |
| --- | --- |
| 视觉层级 | 首屏有一个明确焦点；主操作比装饰更显著；页面不似通用 SaaS |
| 隐私 | 私密内容有持久文字范围；不存在 blur/剪影/元数据泄露 |
| 权威状态 | 线索、投票、淘汰、技能、世界、结果都等服务器确认 |
| 状态完整 | Loading、Empty、Error、Success、Disabled、Offline、Reconnecting 可复现 |
| 响应式 | 360x800、768x1024、1024x768、1440x900、1920x1080 无裁剪/遮挡 |
| 键盘 | 主流程无鼠标可完成；Focus 在场景/Overlay 上清晰 |
| 动效 | 有业务目的；可中断；`prefers-reduced-motion` 有等价终态 |
| 性能 | 无持续重排；图片稳定尺寸；低性能有静态 fallback |
| 文案 | 真实动作、真实后果、合法下一步；无假数据和功能说明式占位 |
| 模式识别 | 截图不依赖标题也能识别三种模式 |

### 13.2 逐页验收

| 页面 | 验收条件 |
| --- | --- |
| 首页 | 10 秒内可理解三模式；继续游玩 ≤ 2 次点击；未开放模式不可产生死点击；单区加载失败不阻塞整页；移动顺序为继续 → 模式 → 推荐 |
| 登录注册 | 登录后恢复白名单 returnTo/邀请；错误不泄露账号存在性；重复提交被阻止；移动输入与错误提示不被键盘遮挡 |
| 游戏大厅 | 筛选写入 URL 且刷新保持；私人房间不出现；快速加入失败保留筛选；空结果显示清除筛选；移动筛选 Drawer 可键盘关闭 |
| 剧本详情 | 无权用户不获得私密正文/真相；CTA 在可用性检查前不可误点；无评论仍有评分入口；下架内容不泄露正文 |
| AI 工坊 | 文件扫描 clean 前不能启动；权利声明缺失不能提交；上传失败可重试/续传；步骤摘要与实际配置一致；移动三步可完成 |
| 解析进度 | 展示当前、完成、失败、重试、人工审核与最终产物；离开页面继续运行；SSE 断开保留最后确认节点；禁止单一无限 Loading |
| 编辑器 | 桌面四区可用；对象树、问题、来源互相定位；阻断问题禁止发布；冲突不静默覆盖；角色预览不泄露其他角色秘密；移动只读/轻编辑 |
| 创建房间 | 人数/版本/规则不合法时不能创建；重复提交不生成双房间；密码不回显；创建成功进入等待且邀请可用 |
| 房间等待 | 席位变化实时；所有条件满足前开始禁用；房主不能看他人秘密；房主迁移与离线正确；聊天入口有明确已接入/未接入状态 |
| 剧本杀 | 当前阶段、计时、主操作固定可见；私密/公共线索有范围；搜证冲突不扣失败方次数；聊天草稿在失败时保留；重连重建快照 |
| 狼人杀 | 席位环/紧凑席位清晰；夜间行动只显示合法目标；提交不提前揭示；死亡不泄露身份；投票/PK/技能链来自规则；AI 降级不阻塞 |
| RPG | 场景第一信号；地图、队伍、任务有抽屉/列表；自由行动先解析再确认；场景版本冲突有替代动作；公共与个人事件分区；NPC 世界动作走规则接口 |
| NPC 对话 | 在场与可见范围明确；私聊/场景对话区分；关系变化需事件依据；世界动作二次确认；AI 安全替换不泄露；正式历史不包含推理链 |
| 结算 | 规则结果先于 AI 点评；结果触发条件可读；AI 失败不阻塞；分享卡片只含脱敏信息；异常终止有事故状态而非猜测结果 |
| 复盘 | 时间线与权威日志一致；角色视角切换重新授权；私聊不自动公开；AI 标注引用事件；移动可折叠阶段；缺失事件显式标记 |
| 个人中心 | 本人/他人视图隔离；隐私保存影响后续接口；昵称/头像审核状态清楚；无成就/作品有合法下一步；移动 Tab 可操作 |
| 游戏历史 | 隐私和历史权限正确；下架内容仅复盘；进行中先校验房间；空列表有大厅入口；完整筛选接口未接入时明确状态 |
| 内容社区 | 游客可读公开内容；含剧透默认折叠；评分需完成对局；重复评分不重复计分；来源/AI 比例/审核状态清楚 |
| 运营后台 | 敏感字段默认脱敏；查看需审批与理由；审核/下架有二次确认和审计；事故排查可按 ID 串联；不直接变更进行中状态 |

### 13.3 三模式专项验收

#### 剧本杀

- 4/6/8 人房间的席位、角色、搜索、投票不产生布局断裂。
- 角色阅读、搜证、讨论、投票、结算的主操作都在对应阶段可用。
- 场景热点与列表等价；独占线索争抢有真实失败反馈。
- 证据板关系必须来自 `known_relationships` 或已知事实。
- DM 生成不影响搜证/投票。

#### 狼人杀

- 6/9/12 人席位环在桌面与移动端可辨。
- 夜间身份、技能、目标、结果只出现在本人授权层。
- 白天发言队列、投票进度、出局与遗言状态清楚。
- 昼夜切换不延迟规则动作，不泄露隐藏角色。
- AI 玩家“思考中”不展示内部推理。

#### RPG

- 场景、地图、队伍、任务、背包在不同断点均可访问。
- 地图节点状态 `current/discovered/reachable/unknown/blocked` 非颜色单示。
- 自由文本解析有候选/澄清/拒绝三种反馈。
- NPC 对话可见范围与关系变化可理解。
- 世界变化有持久状态、事件引用和复盘入口。

### 13.4 组件验收

| 组件 | 验收 |
| --- | --- |
| GameRoom | 阶段、连接、主区、侧区、移动 Tab 稳定；重连不清空合法内容 |
| PlayerCard | 未授权角色不泄露；存活、连接、发言、投票、托管有文字状态 |
| CharacterCard | 私密正文只在授权；Reveal 可跳过；角色预差不影响编辑数据 |
| ClueCard | 内容来源、范围、关联事实可读；获得只在确认事件后播放 |
| EvidencePanel | 列表始终可用；比较/置顶不改变数据真值；私密线索不进入公共分享 |
| RelationshipGraph | 节点/边有真实来源；列表替代可完成选择；未发现关系不暴露数量 |
| Timeline | 创作时间线与对局事件区分；事件缺失不补造；回放不劫持滚动 |
| NPCDialogue | 在场/范围/动作确认；流式失败不污染正式历史 |
| StoryScene | 有明确焦点与环境层；资源失败有 fallback；热点有键盘等价 |
| GamePhaseIndicator | 客户端 0 秒只等待；阶段文本与合法动作来自权威 |
| VotingPanel | 选择、提交、确认、改票、揭票有独立状态；票型不提前泄露 |
| MapExplorer | 路线预览与移动确认；列表可键盘导航 |
| QuestPanel | 状态、可见性、进度、时限清楚；空状态提供探索路径 |
| GenerationWorkflowPanel | 节点状态完整；重试/人工审核/终态可操作；进度不是自增动画 |

### 13.5 视觉验收证据

最终实现阶段需要在浏览器中记录：

- 每个页面的桌面截图：1440x900。
- 关键页面移动截图：360x800 与 390x844。
- 首页、工坊进度、三模式游戏、线索揭示、投票结果、真相页、隐藏结局。
- Loading、Empty、Error、Offline、Reconnecting 至少各一组关键页面证据。
- Reduced Motion 线索/阶段/结局静态终态。
- 证据图/背景图失败 fallback。
- 键盘 Focus 与 Dialog/Sheet 打开/关闭。
- 控制台错误和失败资源检查。

本阶段只交付设计文档，没有运行应用，因此不声明截图、交互或 Canvas/资源渲染已验证。

### 13.6 交付与实现边界

本轮不编写业务代码，不修改 PRD、技术、数据库或 API 文档。

实现阶段建议目录：

```text
apps/web/src/
├── design-system/
│   ├── tokens.css
│   ├── themes.css
│   ├── primitives/
│   └── motion/
├── components/
│   ├── game/
│   ├── evidence/
│   ├── narrative/
│   ├── workflow/
│   └── admin/
├── features/
│   ├── home/
│   ├── lobby/
│   ├── workshop/
│   ├── room/
│   ├── murder/
│   ├── werewolf/
│   ├── rpg/
│   ├── result/
│   └── community/
├── realtime/
├── queries/
└── stores/
```

优先级：

- P0：首页、大厅、Auth、剧本杀工坊/进度/编辑器、房间、等待房聊天、剧本杀、结算/复盘、隐私与重连、最小运营审核后台。
- P1：狼人杀、RPG、NPC、个人中心、历史、社区。
- P2：后台高级工单与审批体验、高级素材、声音、关系图高级空间布局。

### 13.7 待确认事项

这些事项不阻塞设计，但实现前需产品或后端确认：

1. Tailwind 具体版本（配置式或 CSS-first）。
2. 首页推荐/精选/通知/全站搜索的聚合 API。
3. 等待房聊天的精确保留时长、审核策略和举报证据周期（见 OQ-003）。
4. 完整游戏历史与成就 API。
5. 后台运营 API 和敏感字段审批模型。
6. 声音是否进入 MVP，或仅提供设计占位与设置。
7. 角色立绘、剧本封面和证据图的实际内容生产管线。
8. 狼人杀/RPG 的具体 Phase 发布范围与内容首批资产。

---

## 14. 最终设计检查表

- [ ] 共享品牌系统：Archive / Threshold / Thread / Seal 清晰且不滥用。
- [ ] 三模式构图、材料、信息模型可区分。
- [ ] 所有页面与路由已覆盖，含后台。
- [ ] Token 可映射到 Tailwind，业务无魔法色值。
- [ ] 14 个核心组件有职责、Props、状态、事件、动画、API 来源。
- [ ] AI 工作流展示节点、并行、失败、重试、人工审核和终态。
- [ ] 九类关键剧情事件有画面、构图、主体、背景、光影、UI、动画、触发时机。
- [ ] 十类 Motion 契约有触发、时长、缓动、初末、跳过、中断、Reduced Motion。
- [ ] 关键页面覆盖 Loading、Empty、Error、Success、Disabled、Offline、Reconnecting。
- [ ] 实时事件与权威状态分离，序号、快照、重连、幂等可恢复。
- [ ] 私密/公开/阵营/发现者范围有多重视觉与语义信号。
- [ ] 响应式不是缩小桌面，移动主操作持续可见。
- [ ] 无障碍、键盘、焦点、读屏、空间替代、声音/动效设置完整。
- [ ] 图片资源有清单、Prompt、尺寸、fallback 与性能层级。
- [ ] 页面验收标准可自动化或人工复现。
- [ ] 未把设计文档误表述为已实现页面或已验证截图。

