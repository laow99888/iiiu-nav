# iiiu-nav 运维手册

本文面向首次部署和后续维护。命令示例使用 Docker Compose v2；项目只支持部署在域名根路径，不保证子路径部署。

## 1. 运行模型

- 一个容器、一个进程、一个管理员，无注册和子账户。
- 容器以 UID `65532` 的非 root 用户运行，默认监听容器内 `:8080`。
- 所有持久数据都在 `/data`：SQLite 数据库、上传图片和备份归档。
- SQLite 使用 WAL、外键、严格表和前向版本迁移；时间以 UTC Unix 毫秒保存。
- 生产镜像支持 `linux/amd64` 和 `linux/arm64`。

持久目录结构：

```text
/data/
  nav.db
  nav.db-wal
  nav.db-shm
  uploads/
    logos/
    backgrounds/
    site/
  backups/
```

不要只复制 `nav.db` 作为在线备份，也不要手工修改 WAL/SHM 文件。使用管理界面的完整备份功能。

## 2. 首次部署

### 2.1 准备密码和配置

在仓库根目录执行：

```powershell
New-Item -ItemType Directory -Force secrets | Out-Null
[IO.File]::WriteAllText((Join-Path $PWD "secrets/admin-password"), "请替换为至少12个字符的初始密码", [Text.UTF8Encoding]::new($false))
Copy-Item .env.example .env
```

Linux 主机可改用：

```bash
install -d -m 700 secrets
printf '%s' '请替换为至少12个字符的初始密码' > secrets/admin-password
chmod 600 secrets/admin-password
cp .env.example .env
```

密码长度必须为 12-1024 字节。尾部换行会被忽略。`secrets/`、`.env` 和 `/data` 已排除在 Git 与 Docker 构建上下文之外。

### 2.2 构建并启动

```powershell
docker compose build --build-arg VERSION=local
docker compose up -d
docker compose logs app
```

日志应包含 `administrator created`、`data store ready` 和 `iiiu-nav listening`。检查服务：

```powershell
Invoke-WebRequest http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/api/health
```

首次管理员创建成功后，应用不会再次读取初始密码文件。可以将它移到离线密码库或安全删除；如果继续使用本仓库的 Compose 文件，保留一个权限受限的占位文件即可。

Compose 默认：

- 只发布到 `127.0.0.1:${IIU_NAV_PORT:-8080}`；
- 使用 `IIU_NAV_DATA_VOLUME` 指定的命名卷，默认是 `iiiu-nav-data`；
- 根文件系统只读，只有 `/data` 和受限 `/tmp` 可写；
- 丢弃 Linux capabilities 并启用 `no-new-privileges`；
- 异常退出自动重启。

## 3. HTTPS 反向代理

公网访问必须启用 HTTPS。代理必须保留公开 `Host`，并设置 `X-Forwarded-Proto: https`，否则同源写请求和 Secure Cookie 判断会失败。

Caddy 示例：

```caddyfile
nav.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Nginx 示例：

```nginx
server {
    listen 443 ssl http2;
    server_name nav.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

应用只信任 `X-Forwarded-Proto` 来判断公开协议，不使用转发地址绕过登录限速。不要将未受信任客户端直接接入可伪造代理头的内部监听端口。

## 4. 日常备份

登录后打开“管理员菜单 -> 数据备份”：

1. 点击“创建备份”。
2. 等待列表出现新归档。
3. 下载 ZIP 到容器之外的另一块磁盘或备份系统。
4. 定期执行一次恢复演练。

备份使用 SQLite 一致性快照，包含数据库和 `/data/uploads`，清单记录应用版本、schema 版本、文件大小和 SHA-256；管理员会话不会写入归档。创建失败不会发布半成品 ZIP。容器内归档位于 `/data/backups`，但仅保留在同一卷不等于异地备份。

## 5. 完整恢复

恢复会替换数据库和全部上传文件，是破坏性操作：

1. 先下载当前最新备份到容器之外。
2. 打开“管理员菜单 -> 数据备份 -> 恢复完整备份”。
3. 选择 iiiu-nav 生成的 ZIP。
4. 输入当前管理员密码，并输入 `RESTORE`。
5. 提交后等待请求完成，不要重启或终止容器。

系统会先校验 ZIP 路径、大小、清单、哈希、schema、管理员记录和 SQLite 完整性，再自动创建恢复前备份，最后交换数据库与上传目录。任何交换或重新打开失败都会回滚原数据。成功后当前会话失效，必须使用“备份内的管理员密码”重新登录。

拒绝第三方 ZIP、手工修改的归档、比当前应用更新的 schema，以及压缩超过 256 MiB 或解压超过 512 MiB 的归档。

## 6. 修改或找回管理员密码

已登录时直接使用“管理员菜单 -> 修改密码”。忘记密码时，先停止服务，避免运维期间继续写入：

```powershell
docker compose stop app
[IO.File]::WriteAllText((Join-Path $PWD "secrets/admin-password"), "新的至少12个字符密码", [Text.UTF8Encoding]::new($false))
docker compose run --rm app admin reset-password
docker compose up -d
```

Linux：

```bash
docker compose stop app
printf '%s' '新的至少12个字符密码' > secrets/admin-password
chmod 600 secrets/admin-password
docker compose run --rm app admin reset-password
docker compose up -d
```

重置操作在一个事务内更新 Argon2id 哈希并使全部现有会话失效。

## 7. 升级

不要使用未固定的镜像标签直接覆盖生产。升级前记录旧镜像标签或 digest，并完成外部备份。

1. 在管理界面创建并下载完整备份。
2. 记录当前版本：`Invoke-RestMethod http://127.0.0.1:8080/api/health`。
3. 将 `.env` 中 `IIU_NAV_VERSION` 改为明确版本。
4. 构建或拉取新镜像。
5. 执行 `docker compose up -d --build`。
6. 查看 `docker compose logs app`，确认应用版本和 schema 版本。
7. 检查 `/healthz`、公开导航、管理员登录和私有分类。

启动会在监听 HTTP 前只读检查 schema。存在待执行迁移时，应用先在 `/data/backups` 创建完整迁移前备份，再把全部待执行迁移放进一个 SQLite 事务。备份或迁移失败会中止启动，不会提供流量；日志会给出失败迁移和保留的备份名。

## 8. 回滚

如果新版本未改变 schema，可以直接用记录的旧镜像重新启动。若 schema 已升级，旧二进制会明确拒绝新数据库，绝不会尝试降级。

带 schema 回滚的正确顺序：

1. 保留当前数据卷，不要删除或手工改库。
2. 继续使用支持当前 schema 的新版本登录。
3. 在备份界面恢复升级前自动生成的备份；恢复成功后会退出登录。
4. 停止新版本：`docker compose stop app`。
5. 将 `.env` 的 `IIU_NAV_VERSION` 改回已记录的旧版本或 digest。
6. 启动旧版本并检查健康状态、公开导航和管理员登录。

如果新版本已无法启动，但数据卷仍可用，应先用同版本镜像恢复服务，再按上述步骤恢复迁移前备份。最后手段是复制整个卷后，在隔离环境中启动兼容版本并通过网页恢复；不要把备份 ZIP 手工解压覆盖在线 `/data`。

## 9. 多架构镜像

本地构建当前平台：

```powershell
docker build --build-arg VERSION=1.0.0 -t iiiu-nav:1.0.0 .
```

使用 Buildx 构建并推送 amd64/arm64 清单：

```powershell
docker buildx build `
  --platform linux/amd64,linux/arm64 `
  --build-arg VERSION=1.0.0 `
  -t registry.example.com/iiiu-nav:1.0.0 `
  --push .
```

`VERSION` 会写入 `/api/health` 和 OCI 镜像标签，便于确认正在运行的版本。

## 10. 浏览器兼容性

首个版本面向发布时最新两个稳定大版本。2026-08-19 的参考范围：Chrome 151/150、Edge 151/150、Firefox 153/152、Safari 26.5/26.4。官方来源：

- [Chrome Stable 发布记录](https://chromereleases.googleblog.com/)
- [Microsoft Edge 发布计划](https://learn.microsoft.com/en-us/deployedge/microsoft-edge-release-schedule)
- [Firefox 发布记录](https://www.firefox.com/en-US/releases/)
- [Safari 发布说明](https://developer.apple.com/documentation/safari-release-notes)

发布验收使用 Chromium、Firefox 和 WebKit 引擎执行公开页面、搜索、主题、登录对话框及 360/1440 px 响应式烟测。WebKit 烟测不是 macOS 实机 Safari 的替代；正式对外发布前仍应在一台当前 macOS/iOS 设备上复核核心流程。

## 11. 故障排查

```powershell
docker compose ps
docker compose logs --tail 200 app
docker volume inspect iiiu-nav-data
Invoke-WebRequest http://127.0.0.1:8080/healthz
```

常见情况：

- `administrator bootstrap is required`：新数据卷没有管理员，检查 `ADMIN_PASSWORD_FILE` 和 secret 挂载。
- `origin_required` 或 `cross_origin_request`：代理未保留公开 Host，或未设置正确的 `X-Forwarded-Proto`。
- `database schema is newer`：正在使用旧镜像；换回支持该 schema 的镜像，或按回滚章节恢复兼容备份。
- `create pre-migration backup`：磁盘空间或 `/data/backups` 权限不足，修复后再启动，不要绕过备份。
- 上传、导入或恢复被拒绝：检查支持格式和大小限制，不要修改归档来绕过校验。

停止服务前给容器至少 15 秒优雅退出时间。不要同时运行两个面向用户流量的实例共享同一个 SQLite 数据卷。
