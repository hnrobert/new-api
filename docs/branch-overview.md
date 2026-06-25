# 分支概览：相比 main 多了什么

> 分支：`feat/image-aware-model-routing`
> 本文档汇总本分支相对 `main` 引入的全部变更，便于评审与合并前快速了解差异。各项的详细设计见各自链接文档。

本分支在 `main` 基础上新增了 **三项变更**，按性质分组如下：

---

## 1. ✨ 新功能：图片感知模型路由（Image-Aware Model Routing）

提交：`3822eb99 feat(routing): ...`、`b642e05d fix(routing): address review feedback`

配置一个**虚拟入口模型名**（如 `auto-coder`），客户端统一发这个名字，网关在选渠道之前解析请求体，按「最后一条 user 消息是否含图片」自动改写为视觉模型或编程模型。对客户端透明、网关无状态。

- 支持多入口、多轮对话（只看当前轮最后一条 user 消息）
- 可观测性：响应头 `X-Routed-Model` / `X-Route-Entry-Model` / `X-Route-Reason`，Token 级 `ModelRouteNotify` 控制响应体内注入路由提示，日志记录入口模型
- 管理后台「运维设置」提供向导式 Drawer 配置

📄 详细文档：[image-aware-routing.md](./image-aware-routing.md)
📄 PR 描述：[PR-description.md](./PR-description.md)

涉及文件（主要）：`middleware/image_aware_routing.go`、`middleware/distributor.go`、`setting/operation_setting/image_aware_routing.go`、`relay/channel/{claude,openai}/*`、`controller/token.go`、`model/option.go`、前端 `features/system-settings/operations/image-aware-routing-*`。

---

## 2. ✨ 新功能：渠道按用户授权（Per-User Channel Access）

提交：`9380da52 feat: add support for single user channel`

在原有「按分组授权」之外新增正交的「按用户授权」维度：渠道可指定一组 `user_ids`，非空时仅这些用户可用，**无需为单个用户单独建分组**。非授权用户在选渠道、`/v1/models`、`/api/user/models`、`/api/pricing` 四条链路都被屏蔽。

- 存储：`Channel.UserIds`（逗号分隔），不新建表、不改 `abilities` 结构
- 跨库兼容（SQLite/MySQL/PostgreSQL）：user 过滤全部在 Go 里完成，不使用 SQL 字符串包含函数
- 前端 default / classic 两套界面均支持用户多选（异步搜索 + 手填 ID）

📄 详细文档：[per-user-channel-access.md](./per-user-channel-access.md)

涉及文件（主要）：`model/channel.go`、`model/ability.go`、`model/channel_cache.go`、`service/channel_select.go`、`middleware/distributor.go`、`controller/{channel,model,user,pricing}.go`、前端 `features/channels/components/user-multi-select.tsx` 等。

---

## 3. 🔧 CI/CD：Docker 镜像构建与发布流水线

提交：`05c67aa9`、`a93de32a`、`f32612bd`、`47017074`（均为 `ci(docker):`）

- 新增 dev 镜像发布 workflow 与 deploy compose
- dev 镜像按架构在原生 runner 上并行构建
- dev 镜像发布到 **GHCR**（替代 Docker Hub）
- 新增**内嵌前端的全量镜像**构建（单端口 `:3000`，前端打包进二进制）

涉及文件：`.github/workflows/*`、`docker-compose.deploy.yml`。

---

## 与 main 的差异统计

```text
50 个文件改动，+1997 / -59 行
```

覆盖：后端 Go（relay/middleware/model/controller/service/setting）、前端 default + classic、CI workflow、文档。

## 验证状态

- 后端：`go build ./...` 通过；`go vet` 通过；`go test ./model/ ./controller/ ./middleware/ ./service/` 全绿（含两个特性各自的表驱动单测）。
- 图片感知路由：本地实测带图请求走视觉模型、纯文本走编程模型，日志与计费按真实模型。
- 渠道按用户授权：本地 SQLite + mock 上游端到端实测，DB 路径与内存缓存路径行为均符合预期（详见各自文档的「验证」小节）。
