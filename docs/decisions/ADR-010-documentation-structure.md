# ADR-010：文档结构与权威来源

- 状态：Accepted
- 日期：2026-11-21
- 影响范围：全部设计、审查、任务和开发入口文档

## 决策

正式 PRD 路径固定为：

```text
docs/MysteryVerse-PRD.md
```

不创建重复的 `docs/PRD.md`。现有全部 PRD 内容保留并继续作为产品需求源。

## 文档结构

```text
docs/
├── MysteryVerse-PRD.md
├── TECH_DESIGN.md
├── DATABASE_DESIGN.md
├── API_DESIGN.md
├── UI_DESIGN.md
├── TASKS.md
├── DESIGN_REVIEW.md
├── architecture/
├── api/
├── database/
└── decisions/
```

## 权威关系

- PRD 定义产品范围和业务目标。
- ADR 定义已接受的技术决策。
- `architecture/` 定义跨服务边界和不可变架构规则。
- `api/` 定义公共、实时和内部协议基线。
- `database/` 定义逻辑 schema、事件模型和数据不变量。
- `TECH_DESIGN.md`、`DATABASE_DESIGN.md`、`API_DESIGN.md`、`UI_DESIGN.md`
  保留详细设计；与 ADR 冲突时必须修订。
- `TASKS.md` 只定义可执行任务，不得创造与 ADR 冲突的替代设计。
- `DESIGN_REVIEW.md` 保留历史审查；新增解决结论记录在 ADR 和审查附录。

## 维护规则

- API 变化同步 `api/`、`API_DESIGN.md` 和 `packages/contracts` 任务。
- 数据变化同步 `database/`、`DATABASE_DESIGN.md` 和迁移任务。
- 跨服务边界变化必须新增或更新 ADR。
- 不允许创建空文档占位。
- 不允许为了匹配目录而复制长篇内容；新文档应引用权威长文档并只定义必要规范。
- 所有相对链接必须可解析。

## 影响

- Codex 开工入口固定为 `AGENTS.md`、`architecture/system-architecture.md`、
  相关 ADR 和目标任务。
- `docs/PRD.md` 路径问题永久关闭。

## 关联文档

- [系统架构](../architecture/system-architecture.md)
- 根 `AGENTS.md`
- `TASKS.md` §0.1

