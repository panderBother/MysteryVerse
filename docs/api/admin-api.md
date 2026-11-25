# 运营后台 API

> 状态：Candidate Baseline
> 根路径：`/api/v1/admin`
> 发布门禁：公开 MVP
> 依据：ADR-007

## 1. 权限

后台使用独立 Admin Session 和 RBAC/ABAC：

- `content_reviewer`：内容审核和发布状态，不读取玩家私聊。
- `support_agent`：举报、事故摘要和有限用户处置。
- `security_auditor`：审计和敏感访问审批。
- `platform_admin`：全局管理，关键操作可要求双人复核。

所有非 GET 请求：

- 使用 CSRF/Origin 校验。
- 必须带 `Idempotency-Key`。
- 记录操作理由。
- 写不可变审计。

## 2. 内容审核与发布

```text
GET  /api/v1/admin/scripts/review-queue
GET  /api/v1/admin/scripts/{script_id}/versions/{version_id}
POST /api/v1/admin/scripts/{script_id}/versions/{version_id}/moderate
POST /api/v1/admin/scripts/{script_id}/versions/{version_id}/publish
POST /api/v1/admin/scripts/{script_id}/versions/{version_id}/unpublish
```

审核决策：

```json
{
  "decision": "approve|reject|request_changes",
  "reason_code": "CONTENT_POLICY",
  "reason": "存在未标注的侵权内容",
  "expected_row_version": 7
}
```

发布必须复用 Go 原子发布事务。后台不得直接改公开指针绕过校验。

## 3. 举报

```text
GET  /api/v1/admin/reports
GET  /api/v1/admin/reports/{report_id}
POST /api/v1/admin/reports/{report_id}/actions
```

支持：

- 受理、驳回、升级、证据保全、下架、转事故。
- 定位消息、事件、版本和操作者。
- 私密正文只在授权 scope 内读取并记录审计。

## 4. 用户处置

```text
POST /api/v1/admin/users/{user_id}/sanctions
POST /api/v1/admin/sanctions/{sanction_id}/revoke
GET  /api/v1/admin/users/{user_id}/sanctions
```

处罚请求：

```json
{
  "sanction_type": "mute|suspend",
  "reason_code": "HARASSMENT",
  "reason": "多次骚扰其他玩家",
  "scope": "platform",
  "expires_at": "2026-12-01T00:00:00Z"
}
```

处罚与撤销必须幂等、审计，并在下一次请求和 WebSocket 写操作时生效。

## 5. 事故

```text
GET  /api/v1/admin/incidents
GET  /api/v1/admin/incidents/{incident_id}
POST /api/v1/admin/incidents/{incident_id}/actions
```

允许冻结游戏、停止广播、启动恢复或安全终止。事故操作不直接改历史事件。

## 6. AI 生成内容

```text
GET  /api/v1/admin/generation/review-queue
POST /api/v1/admin/generation/{job_id}/actions
```

支持查看已脱敏候选摘要、执行接受/拒绝、阻止发布和记录原因。原始 D4 只按审批
scope 获取。

## 7. 审计

```text
GET /api/v1/admin/audit-logs
GET /api/v1/admin/audit-logs/{audit_id}
```

审计日志不可变。普通管理员不能修改或删除历史。

## 8. 公开 MVP 最低门禁

- 上架/下架可用。
- 内容审核可用。
- 举报可定位和处理。
- 禁言/封禁可执行和撤销。
- 事故可查询和冻结。
- AI 内容可审核。
- 审计可查询。

缺少任一项时，仅允许内测发布。

