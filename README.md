# ipa-renamer

跨平台 IPA 重命名工具：扫描（或持续监听）输入目录下的 `.ipa` 文件，读取其 `Payload/<App>.app/Info.plist` 中的 `CFBundleIdentifier`（直接在 zip 内读取并放入内存解码），然后把文件**复制**到输出目录，命名为

```
<原文件名>@<CFBundleIdentifier>.ipa
```

源码位于 [`src/`](src/) 目录，按职责拆分为独立模块。仓库同时提供多阶段 `Dockerfile`（容器启动参数通过环境变量注入，由入口脚本转换为命令行参数）以及两套相互独立的 GitHub Actions workflow：一套发布 release 产物，一套构建并推送容器镜像。

## 使用方式

```
ipa-renamer [options] [INPUT]
```

参数：

| 参数 | 说明 |
| --- | --- |
| `INPUT` | 输入目录，扫描其中的 `.ipa`；缺省为当前工作目录 |
| `-i, --input <DIR>` | 输入目录，与位置参数 `INPUT` 等价（两者同时给出且值不同时报错） |
| `-o, --output <DIR>` | 输出目录，存放重命名后的文件（缺省 `renamed`） |
| `-t, --time <秒>` | watch 模式下，文件持续多少秒无变动才触发处理（缺省 5） |
| `-w, --watch` | 监听输入目录：持续处理新增/变动的 `.ipa`，而不是扫描一次后退出 |
| `-h, --help` | 显示帮助 |
| `-V, --version` | 显示版本 |

示例：

```sh
# 扫描 /path/to/input 下的 .ipa
ipa-renamer /path/to/input -o /path/to/output

# 与上面等价，使用 -i 形式
ipa-renamer -i /path/to/input -o /path/to/output

# 扫描当前目录
ipa-renamer -o /path/to/output

# 监听输入目录，文件静默 8 秒后处理
ipa-renamer -w -t 8 -i /path/to/input -o /path/to/output
```

### 监听（watch）模式细节

- `-w` 为布尔开关，监听对象即输入目录。
- 启动后先处理目录中已存在的 `.ipa`，之后监听新增/写入/重命名事件。
- 防抖：某文件持续 `-t/--time` 秒（缺省 5）没有新的文件事件才触发处理，避免在上传过程中就重命名。
- 已符合 `<名称>@<bundle-id>.ipa` 命名规则的文件会被跳过。

### 参数默认值

默认值直接编译在程序中，作为未显式传参时的兜底：

| 参数 | 缺省值 |
| --- | --- |
| 输入目录 | 当前工作目录 |
| 输出目录 | `renamed`（相对当前工作目录） |
| 防抖时间 | 5 秒 |

程序本身不读取环境变量；环境变量仅用于容器入口脚本在启动时构造命令行参数。

## 本地构建与测试

需要 Go 1.27+。

```sh
# 构建（源码在 src/ 下）
go build -o ipa-renamer ./src

# 版本注入（release 也使用同样方式）
go build -ldflags "-X github.com/ofwh/ipa-renamer/src.version=1.2.3" -o ipa-renamer ./src

# 测试 / 静态检查
go test ./src/...
go vet ./src/...
```

## Docker

镜像分两阶段构建：

- **builder**：`golang:1.27-alpine`，`CGO_ENABLED=0` 静态编译。
- **runtime**：`alpine:3.24`，包含静态二进制与入口脚本 [`docker/entrypoint.sh`](docker/entrypoint.sh)。

容器参数**不通过 `command` 指定**，而是以环境变量（`.env` / `environment`）传入；入口脚本读取 `IPA_RENAMER_*` 变量，拼装成命令行参数后启动程序：

| 环境变量 | 对应参数 | 缺省 |
| --- | --- | --- |
| `IPA_RENAMER_WATCH` | `-w`（设为 `0`/`false`/`no`/`off` 则关闭） | 开启 |
| `IPA_RENAMER_INPUT` | `-i` | 程序内置缺省 |
| `IPA_RENAMER_OUTPUT` | `-o` | 程序内置缺省 |
| `IPA_RENAMER_TIME` | `-t` | 程序内置缺省 |

```sh
# 构建
docker build -t ipa-renamer .

# 运行：监听 /app/in，输出到 /app/out
docker run \
  -e IPA_RENAMER_WATCH=1 \
  -e IPA_RENAMER_INPUT=/app/in \
  -e IPA_RENAMER_OUTPUT=/app/out \
  -v "$PWD/watched:/app/in" -v "$PWD/output:/app/out" \
  ipa-renamer
```

镜像默认以非特权用户 `65534`（nobody）运行，工作目录为可写的 `/data`。宿主 bind mount 目录的所有者通常是本机用户，直接挂载可能遇到权限问题：可将挂载目录 `chown -R 65534:65534`，或在运行时用 `--user 0` 覆盖。

### docker-compose

```sh
cp .env.example .env   # 按需修改参数
docker-compose up -d
```

[`docker-compose.yml`](docker-compose.yml) **直接引用 GHCR 镜像**（`ghcr.io/ofwh/ipa-renamer:latest`，由 docker-image workflow 推送），不使用本地 `build`；参数通过 `environment` 注入（缺省值取自同目录 `.env`，见 [`.env.example`](.env.example)），并挂载 `./watched`、`./output` 两个目录。

## 目录结构

```
.
├── src/                          # Go 源码（package main）
│   ├── main.go                   # 入口：分派到 one-shot / watch
│   ├── cli.go                    # 参数解析（-h/-i/-o/-t/-w/-V、位置参数、内置缺省）
│   ├── config.go                 # 目录解析与创建
│   ├── renamer.go                # 复制/命名核心、命名规则、已命名判定
│   ├── ipa.go                    # 定位并解码 Info.plist，取 CFBundleIdentifier
│   ├── watcher.go                # fsnotify 监听 + 防抖（-t/--time）+ 初始扫描
│   ├── log.go                    # 分级日志（INFO 到 stdout，WARN/ERROR 到 stderr）
│   └── version.go                # 版本号（可由 ldflags 注入）
├── docker/
│   └── entrypoint.sh             # 入口脚本：IPA_RENAMER_* 环境变量 → 命令行参数
├── Dockerfile                    # 多阶段镜像（builder: golang; runtime: alpine）
├── docker-compose.yml            # 直接使用 GHCR 镜像，参数经 environment 注入
├── .env.example                  # 容器参数示例（复制为 .env 使用）
├── .github/workflows/
│   ├── release.yml               # tag v* → 多平台二进制 + GitHub Release
│   └── docker-image.yml          # main/tag → 多架构镜像推送 GHCR
└── ...
```

## CI/CD

- **[`release.yml`](.github/workflows/release.yml)**：推送 `v*` 标签（或手动触发）时，在 ubuntu 上一次交叉编译 linux/darwin/windows × amd64/arm64，打包 `tar.gz` 与 `sha256`，发布到 GitHub Release。版本号经 `-X .../src.version=` 注入二进制。
- **[`docker-image.yml`](.github/workflows/docker-image.yml)**：推送 `main`/`master` 或 `v*` 标签（或手动触发）时，构建 `linux/amd64`、`linux/arm64` 双架构镜像并推送到 `ghcr.io/<owner>/<repo>`：
  - 分支推送：`:latest`
  - 标签推送：额外打 `:vX.Y.Z` 与 `:X.Y.Z`

依赖版本：

| 组件 | 版本 |
| --- | --- |
| Go | 1.27 |
| builder 镜像 | `golang:1.27-alpine` |
| runtime 镜像 | `alpine:3.24` |
| fsnotify | v1.10.1 |
| howett plist | v1.0.0 |
| actions: checkout / setup-go | v7 / v6 |
| docker actions（qemu/buildx/login） | v4 |
| docker/build-push-action | v7 |
| softprops/action-gh-release | v3 |

## License

[MIT](LICENSE) © 2026 ofwh
