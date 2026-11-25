# ADR-008：PlayerContext 构建机制

- 状态：Proposed
- 日期：2026-11-21
- 影响范围：AI DM、NPC、玩家上下文、Context Resolve、Memory、安全

## 决策

MVP 每次 Agent 执行前从当前可见投影重建 PlayerContext，不持久化
PlayerContext 快照。

PlayerContext 至少包含：

- 当前 `game_id`、`state_version`、`event_seq`。
- 当前阶段、截止时间和服务端时间。
- 玩家、角色、席位和 actor scope。
- 当前可用动作。
- 玩家可见的公共状态和个人状态。
- 允许访问的历史事件或事件摘要。
- 当前 Memory View 或短期上下文摘要。

## 构建流程

1. Go 校验 Session、房间成员、游戏席位、角色绑定和阶段。
2. Go 生成 `AllowedKnowledgeSlice` 与 `PlayerContext` 授权信封。
3. Python 只能在切片内做排序、摘要、token 裁剪和格式组装。
4. Python 不得根据实体 ID 补查 Go 数据。
5. Go 在发送给模型前再次校验版本、安全策略和输出约束。
6. `context_manifest` 记录授权实体、版本、裁剪结果和哈希，不保存完整秘密正文。

## 隔离

- 玩家只能获得自己有权查看的角色本、私密线索和私聊。
- DM 不因主持身份自动拥有玩家秘密。
- NPC 只能获得自身可见知识和授权 Memory View。
- 任何跨玩家共享缓存必须包含 actor scope、sensitivity 和 version。
- 本地内存、Redis 或对象缓存均不得绕过 Go 授权。

## 扩展

未来可增加上下文快照或缓存，但必须：

- 使用独立 ADR。
- 定义作用域、版本、过期、加密和来源事件。
- 保证缓存丢失可从 Go 权威投影重建。
- 不得把 PlayerContext 变成新的权威业务状态。

## 关联文档

- [Memory 架构](../architecture/memory-architecture.md)
- [服务边界](../architecture/service-boundaries.md)
- [内部 AI API](../api/ai-internal-api.md)

