# 安全与权限

> 状态：DQ-19 安全基础 APPROVED / IMPLEMENTED / TEST-VERIFIED；生产基础设施
> NOT VERIFIED。其他权限细节仍依赖 Proposed ADR。
> 依据：ADR-001、ADR-002、ADR-004、ADR-006、ADR-007、ADR-008

## 1. 数据分级

| 级别 | 示例 | 传输与缓存 |
| --- | --- | --- |
| D0 | 公开剧本简介、封面、标签 | 可公开缓存 |
| D1 | 房间成员、公开阶段、公聊、公共线索 | 当前房间/对局授权 |
| D2 | 角色本、私人线索、私聊、个人任务 | 加密、仅本人/角色 |
| D3 | 身份分配、狼人阵营、技能结果、随机审计 | Runtime/极小权限 |
| D4 | 完整真相、身份表、事故证据 | 结算后授权或审批 |

## 2. 授权链

```text
Session
→ account status
→ content visibility
→ room membership
→ seat / role binding
→ game ownership
→ object ACL
→ field ACL
→ reveal gate
→ event visibility
→ action permission
→ DTO projection
→ audit
```

任一步失败即拒绝。客户端隐藏不是权限。

## 3. 服务端投影

- REST 和 WebSocket 只返回 DTO。
- 私密事件定向发送。
- 房主、创作者和普通运营没有自动秘密访问权。
- 分享、导出、复盘和搜索必须重新执行 ACL。
- 对象存储 URI 必须短期签名或受控下载。

## 4. AI 安全

- Go 先裁剪知识切片。
- 用户输入作为不可信数据区。
- 工具白名单。
- Prompt、身份表和真相不进入普通上下文。
- 输出经过 Schema、规则和泄漏检查。
- `chain_of_truth`、Prompt、内部 trace 不返回客户端。
- 同步 AI 回合禁止 Python 回调 Go 再触发 AI 的嵌套环。

## 5. 等待房聊天

- 使用 `room_chat` 独立流。
- 仅房间成员可读写。
- 有长度、频率、禁言和审核限制。
- 不改变房间或游戏状态版本。
- 不进入公开 CDN。

## 6. 运营后台

- 独立后台 API、Session 和权限。
- RBAC/ABAC。
- 敏感访问需要理由、scope、时效和审计。
- 高权限操作可要求审批或双人复核。
- 不得复用玩家 API 旁路读取秘密。
- 禁言、封禁、下架和举报处理必须可审计和可撤销。

## 7. 存储

- D2-D4 使用 `SecretService` 和 envelope encryption。
- key id 与 nonce 不返回客户端。
- 大对象使用加密对象存储和 scope 路径。
- Redis key 包含 game、actor、scope、version。
- 私密投影不进入共享缓存。

## 8. 日志与审计

禁止普通日志记录：

- D2-D4 正文。
- Prompt 和完整上下文。
- Token、Cookie、密钥和连接串。
- 完整私聊。

审计记录对象 ID、访问结果、scope、理由和 hash，不记录正文。

## 9. 测试

必须覆盖：

- URL/body 篡改 ID。
- WebSocket 伪造事件/频道/seq。
- 房主、作者和运营越权。
- 私聊、角色本、真相和 Memory。
- Prompt injection 和跨 Agent 注入。
- 对象存储、分享、导出和搜索旁路。
- 旧 Session/ticket 失效。
- 后台敏感访问和审计。

## 10. Agent Runtime 服务身份与请求安全

Task 11 后正式内部安全基础：

- Service ID + 短期 Service Token 作为开发基础认证。
- HMAC-SHA256 `mv-hmac-sha256-v1`，覆盖 signature version、Key ID、timestamp、
  nonce、Service ID、method、path、query 和 body hash。
- Key ID、Active/Previous 轮换、撤销、Key Provider/KMS 抽象。
- nonce 通过可替换 Store 原子消费；生产配置要求 Redis，Store 故障 fail-closed。
- operation/resource/tenant scope ACL，按认证后的 Service Identity 授权。
- 结构化安全审计，不记录 token、签名密钥或业务正文。
- mTLS 服务端配置和 Go 客户端证书链校验；本地/测试允许显式关闭。
- D3/D4 Context 默认不发送模型。
- 外部 MCP 和 external network 工具默认拒绝。

DQ-19 已批准生产目标为 mTLS + HMAC。真实 CA/KMS/Redis 多实例尚未部署，生产
能力仍标记 `NOT VERIFIED`。详细威胁模型见
`docs/architecture/dq-18-22-ai-runtime-design.md`，内部协议见
`docs/api/ai-internal-api.md`。

批准证据和实施边界见 `docs/architecture/task-11-approval-blockers.md`。

