# K8s manifests 配置指南

本文介绍如何使用本仓库内置的 Kustomize manifests 在 Kubernetes 中部署 `prometheus-dingtalk-hook`，以及如何配置 `config.yaml`/模板与热重载。

## 目录结构

- `k8s/base/`：通用资源（Deployment/Service/Secret/ConfigMap）
- `k8s/overlays/dev/`：开发环境 overlay（namespace、少量 patch）
- `k8s/overlays/prod/`：生产环境 overlay（副本数、资源 requests/limits）

## 快速开始

1) 准备一个 namespace（overlay 已配置为 `prometheus-dingtalk`）：

```bash
kubectl create namespace prometheus-dingtalk
```

2) 直接部署（dev overlay）：

```bash
kubectl apply -k k8s/overlays/dev
```

3) 验证：

```bash
kubectl -n prometheus-dingtalk get deploy,po,svc
```

## 服务与端点

容器端口：`9098`。

HTTP 端点：

- `POST /alert`：Alertmanager Webhook（路径可通过 `server.path` 改）
- `GET /healthz`：liveness probe
- `GET /readyz`：readiness probe
- `POST /-/reload`：触发热重载（需要 `reload.enabled: true`，且当前实现未做鉴权）
- `GET/PUT /admin/*`：管理 UI（需要 `admin.enabled: true` 且 BasicAuth 通过）

## 配置文件（Secret）

manifests 默认把配置放在 Secret：`prometheus-dingtalk-hook-config`，key 为 `config.yaml`，挂载到：

- 配置目录：`/etc/prometheus-dingtalk-hook/config.yaml`
- 启动参数：`-config /etc/prometheus-dingtalk-hook/config.yaml`

你需要重点修改的字段：

- `server.listen`：建议保持 `"0.0.0.0:9098"`
- `server.path`：Alertmanager webhook path（默认 `"/alert"`）
- `auth.token`：可选共享 token；支持请求头 `Authorization: Bearer <token>` 或 `X-Token: <token>`
- `dingtalk.robots[].webhook`：必填
- `dingtalk.robots[].secret`：机器人“加签”才需要
- `dingtalk.channels`：必须包含 `name: "default"` 且绑定至少一个 robot
- `dingtalk.routes`：可为空；为空时所有告警走 `default` channel

编辑方式（推荐）：复制 `k8s/base/secret.yaml` 到自己的 overlay，然后在 overlay 中替换内容。

示例（只展示关键点）：

```yaml
dingtalk:
  robots:
    - name: "default"
      webhook: "https://oapi.dingtalk.com/robot/send?access_token=..."
  channels:
    - name: "default"
      robots: ["default"]
```

## 模板（ConfigMap）

manifests 提供一个可选 ConfigMap：`prometheus-dingtalk-hook-templates`，默认包含 `default.tmpl`。

挂载目录：`/etc/prometheus-dingtalk-hook/templates`。

配置项：

```yaml
template:
  dir: "/etc/prometheus-dingtalk-hook/templates"
```

说明：

- 程序始终内置一个 `default` 模板；即使目录为空/不存在也会回退到内置模板。
- 若你在 `template.dir` 中放置 `xxx.tmpl`，模板名就是 `xxx`。
- `channels[].template` 需要引用存在的模板名（默认 `default`）。

如果你不想挂载模板 ConfigMap：

- 删除 Deployment 中的 `templates` volume/volumeMount
- 把 `template.dir` 设为空字符串（或指向不存在目录），程序会使用内置 `default`

## 热重载

当前实现支持两种方式触发 reload：

1) 定时轮询（`reload.enabled: true` + `reload.interval`）
2) HTTP 触发：`POST /-/reload`

重载检测范围：

- 配置文件：基于 `stat(size, mtime)`
- 模板目录：对 `*.tmpl` 文件同样基于 `stat(size, mtime)`

Kubernetes 注意事项：

- 使用 ConfigMap/Secret volume 时，更新通常会通过“原子替换”更新文件/目录；轮询模式一般可感知。
- 如果你不启用轮询，可以在更新 ConfigMap/Secret 后，手动 `POST /-/reload`。
- `/-/reload` 目前未做鉴权：若暴露到集群外，请务必通过 Ingress/NetworkPolicy 限制访问。

## 管理 UI（可选）

启用：

```yaml
admin:
  enabled: true
  path_prefix: "/admin"
  basic_auth:
    username: "admin"
    password: "change-me"
```

说明：

- UI 会读取并写回 `-config` 指定的配置文件路径（因此配置卷需要可写）。
- 由于本 manifests 将配置挂载为 Secret 且只读，默认不适合开启 UI 的“在线修改配置”功能。
- 若确实需要 UI 在线编辑：建议使用 PVC（ReadWrite）保存 `config.yaml` 与模板目录，再调整 volume/volumeMount。

## Alertmanager 配置示例

把 receiver 指向 Service：

```yaml
receivers:
  - name: ops-team
    webhook_configs:
      - url: "http://prometheus-dingtalk-hook.prometheus-dingtalk.svc:9098/alert"
        send_resolved: true
```

## 常见问题

1) Pod 启动失败：`default channel is required`
- `dingtalk.channels` 必须包含 `name: "default"`。

2) 返回 401
- 你启用了 `auth.token`，但请求没有带 token，或 token 不匹配。

3) 模板不生效
- 确认 `template.dir` 指向挂载路径且文件扩展名为 `.tmpl`。
- 确认 `channels[].template` 引用的是模板名（不带 `.tmpl`）。
