# Per-User Channel Access（渠道按用户授权）

## 功能概述

在原有的「按用户分组（group）授权」之外，新增一个正交的「按用户授权」维度：一个渠道除了能限定给某个分组外，还能**限定给指定的若干用户**。

解决的实际痛点：管理员把一个私有 API 配成渠道后，想「只给某个特定用户用」，以前**必须**为该用户单独建一个分组、把渠道挂到该分组、再让该用户能访问该分组——很繁琐。现在只需在渠道里直接选择该用户即可，**无需为单个用户建分组**。

## 配置方式

在「渠道管理 → 编辑渠道」表单的「模型与分组」区域，分组字段下方有一个 **Authorized Users / 指定用户** 字段：

- **留空**（默认）：不限制，沿用原有 group 行为——所选分组内所有用户可用。
- **选择/输入用户**：仅这些用户可用该渠道。支持多选，搜索框按用户名/邮箱/显示名异步搜索（调用 `/api/user/search`），也可直接输入数字用户 ID。

前端两套界面都已支持：
- `web/default`（新版）：基于 MultiSelect 的 `UserMultiSelect` 组件，异步搜索 + 已选用户名自动回填。
- `web/classic`（老版）：Semi `Form.Select` 多选 + `allowAdditions`，支持关键字搜索与手填 ID。

提交时前端把用户 ID 数组用逗号拼成字符串发给后端，存入渠道的 `user_ids` 字段（逗号分隔，空串表示不限制）。

## 行为说明

设了 `user_ids` 后，该渠道对**非授权用户**在以下所有链路都被屏蔽：

| 链路 | 授权用户 | 非授权用户 |
|------|---------|-----------|
| 渠道选择（`/v1/chat/completions` 等 relay） | 可命中该渠道 | 该渠道被过滤掉；若无其他可用渠道则返回 `No available channel` |
| `GET /v1/models`（OpenAI 兼容模型列表） | 列表含该模型 | 该模型被隐藏 |
| `GET /api/user/models`（前端「我的可用模型」） | 含该模型 | 该模型被隐藏 |
| `GET /api/pricing`（定价/倍率页） | 含该模型 | 若该模型所有渠道都对当前用户关闭，则隐藏 |

关键规则：
- **匿名用户（未登录 / 内部调用，userId ≤ 0）对受限渠道一律拒绝**。
- 过滤是**叠加在分组之上**的：渠道仍必须属于用户所在分组，user_ids 只是再收窄一层。
- 若同一模型既有「私有渠道（限定用户）」又有「公共渠道（无限制）」，非授权用户会自动落到公共渠道，体验正常。

## 实现要点（面向开发者）

- **数据模型**：`Channel.UserIds string`（`model/channel.go`，逗号分隔，varchar(1024)）。配套方法 `GetUserIds()`、`IsUserRestricted()`、`IsUserAllowed(userId)`。**不新建表、不改 `abilities` 表结构**——user 限制是渠道级属性，与 model 无关。
- **跨库兼容（SQLite / MySQL / PostgreSQL）**：user 过滤**绝不进 SQL 字符串函数**（`FIND_IN_SET` 等不跨库）。统一策略：
  - 内存缓存路径：用已缓存的 `channelsIDM[id].UserIds` 在 Go 里过滤（`filterChannelsByUser`）。
  - DB 路径：`JOIN channels` 读出 `user_ids` 后在 Go 里过滤（`filterAbilities` 把 path 与 user 过滤合并到一次 channel 加载）。
- **四条链路全部接入 userId**：
  - 选渠道：`GetRandomSatisfiedChannel` / `GetChannel`（两条路径）签名加 `userId`，经 `RetryParam.UserId` 由 `middleware/distributor.go` 透传（取自 `c.GetInt("id")`）。
  - 渠道亲和（affinity）：distributor 里直接用 `preferred.IsUserAllowed(userId)` 判断，不改 `IsChannelEnabledForGroupModel` 签名。
  - 模型列表：新增 `GetGroupEnabledModelsForUser(groups, userId)`，`ListModels` / `GetUserModels` 改用它。
  - pricing：`filterPricingByUserAccess` 在分组过滤之上用 `GetGroupEnabledModelsForUser` 做二次过滤，全局缓存不变。
- **写入与校验**：`validateChannel` 里调 `normalizeUserIds`——去空白、去重、逐个 `Atoi` 校验（非法 / 0 / 负 ID 报错且原值不变），空串归一化为 `""`。
- **缓存同步**：`CacheUpdateChannel` 整指针替换，`UserIds` 自动带，零额外代码；批量改渠道、`FixAbility` 均无需特殊处理（user_ids 在渠道上，不在 abilities）。

## 边界与限制

- **Token 指定渠道路径**（管理员在 URL 里带 `/<channel_id>` 强制指定渠道）：保留原语义，不做 user 限制（管理员强制指定）。
- **auto 分组**：userId 一路透传，无特殊处理。
- 渠道列表页是否展示「指定用户」列：当前未加（数字 ID 可读性差），如需要可作为后续增强。

## 验证

本地用 SQLite + mock 上游实测（alice=id 2 授权、bob=id 3 未授权，同在 `default` 组，渠道 `user_ids=2`）：

```
DB 路径(MEMORY_CACHE_ENABLED=false):
  alice /v1/models        -> ['gpt-4o-mini']
  bob   /v1/models        -> []
  alice chat              -> 200（命中渠道）
  bob   chat              -> No available channel

内存缓存路径(MEMORY_CACHE_ENABLED=true):
  行为与 DB 路径完全一致

混合场景（再加一个无 user_ids 的公共渠道）:
  bob chat -> 200（自动走公共渠道，不碰私有渠道）

编辑渠道更新 user_ids:
  "2, 3 , 2" -> 归一化为 "2,3"，bob 立即放行（缓存同步生效）
  "2, 3 ,abc" -> 报错拒绝，原值不变
```

后端 `go build ./...`、`go vet`、`go test ./model/ ./controller/ ./middleware/ ./service/` 全部通过；新增表驱动单测 `model/channel_user_ids_test.go`、`controller/channel_user_ids_test.go`。
