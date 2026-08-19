# iiiu-nav

一个轻量、单管理员、自托管的个人导航站。公开页面展示分类链接；登录后可在同一界面管理分类、链接、站点外观、书签、备份和恢复。

项目范围、验收标准和开发状态以 [`docs/DEVELOPMENT_PLAN.md`](docs/DEVELOPMENT_PLAN.md) 为准。生产部署、升级、回滚和数据恢复请直接阅读 [`docs/OPERATIONS.md`](docs/OPERATIONS.md)。

## 快速部署

要求 Docker 29+ 和 Docker Compose v2。

```powershell
New-Item -ItemType Directory -Force secrets | Out-Null
[IO.File]::WriteAllText((Join-Path $PWD "secrets/admin-password"), "请替换为至少9个字符的初始密码", [Text.UTF8Encoding]::new($false))
Copy-Item .env.example .env
docker compose build
docker compose up -d
```

打开 `http://127.0.0.1:8080` 查看公开前台；管理员从 `/admin/login` 进入独立后台。Compose 默认只监听本机回环地址；公网部署应通过 HTTPS 反向代理访问。首次启动会读取密码文件并只保存 Argon2id 哈希，后续启动不会再读取该文件。

Linux 上容器以 UID `65532` 运行。若密码文件由 root 创建，启动前还需要执行 `chown root:65532 secrets/admin-password && chmod 640 secrets/admin-password`，否则容器无法读取首次启动密码。

```powershell
Invoke-WebRequest http://127.0.0.1:8080/healthz
docker compose logs app
```

数据保存在命名卷 `iiiu-nav-data`，不要在升级或重建容器时删除该卷。详细的权限、代理、备份、恢复和回滚步骤见[运维手册](docs/OPERATIONS.md)。

## 本地开发

要求 Go 1.26.6、Node.js 22.12+ 和 npm 11+。

```powershell
npm install
$env:ADMIN_PASSWORD_FILE = "C:\secure\iiiu-nav-admin-password"
npm run dev:api
```

另开一个终端：

```powershell
npm run dev:web
```

打开 `http://127.0.0.1:5173`。默认数据目录是 `./data`；可通过 `IIU_NAV_DATA_DIR` 指定其他目录，API 监听地址可通过 `IIU_NAV_ADDR` 修改。每日 PV 按 `IIU_NAV_TIMEZONE` 归属自然日，默认使用 `Asia/Shanghai`。

## 构建与检查

```powershell
npm run check
npm run build
docker build --target go-test -t iiiu-nav:test .
docker build --build-arg VERSION=local -t iiiu-nav:local .
```

生产构建将前端资源嵌入单个 Go 二进制。容器以非 root 用户运行，持久化内容都位于 `/data`。
