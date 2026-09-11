# ipa-renamer

跨平台 IPA 重命名工具：扫描（或持续监听）输入目录下的 `.ipa` 文件，读取其 `Payload/<App>.app/Info.plist` 中的 `CFBundleIdentifier`（直接在 zip 内读取并放入内存解码），然后把文件**重命名**为 `<原文件名>@<CFBundleIdentifier>.ipa`

## 使用方式

```
ipa-renamer [options] [INPUT]
```

参数：

| 参数 | 说明 |
| --- | --- |
| `INPUT` | 输入目录，扫描其中的 `.ipa`；缺省为当前工作目录 |
| `-i, --input <DIR>` | 输入目录，与位置参数 `INPUT` 等价（两者同时给出且值不同时报错） |
| `-o, --output <DIR>` | 输出目录，存放重命名后的文件（缺省与输入目录相同） |
| `-c, --copy` | 复制重命名后的文件，保留源 `.ipa`；缺省为重命名，源文件会被删除 |
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

## 本地构建

需要 Go 1.27+。

```sh
# 构建（源码在 src/ 下）
go build -o ipa-renamer ./src

# 版本注入（release 也使用同样方式）
go build -ldflags "-X main.version=1.2.3" -o ipa-renamer ./src

# 静态检查
go vet ./src/...
```

## Docker

```sh
# 构建
docker build --build-arg VERSION=0.1.0 -t ipa-renamer .

# 运行：监听 /app/in，输出到 /app/out
docker run -v "$PWD/watched:/app/in" -v "$PWD/output:/app/out" ipa-renamer
```

镜像默认以非特权用户 `65534`（nobody）运行，工作目录为 `/app`。宿主 bind mount 目录的所有者通常是本机用户，直接挂载可能遇到权限问题：可将挂载目录 `chown -R 65534:65534`，或在运行时用 `--user 0` 覆盖。

### docker-compose

- 复制 [`docker-compose.yml`](docker-compose.yml) 按需修改（主要是 `user` 部分）
- 复制 [`.env.example`](.env.example) 为 `.env` 后修改

```sh
cp .env.example .env
docker-compose up -d
```

## License

[MIT](LICENSE) © 2026 ofwh
