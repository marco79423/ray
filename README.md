# ray

SBK 後端小幫手

## 安裝

```shell
go install github.com/marco79423/ray/bin/ray
```

## 使用

```shell
ray --help
NAME:
   ray - SBK 後端小幫手

USAGE:
   ray [global options] command [command options] [arguments...]

COMMANDS:
   publish, worship, p, fuck  升版
   help, h                    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help (default: false)
```

### 發布版本

```shell
NAME:
   ray publish - 發布版本

USAGE:
   ray publish [command options] [版號]

OPTIONS:
   --keyfile value, -f value  Private Key 檔案路徑
   --keyfile-password value   Private Key 的密碼
   --path value, -p value     Repository 路徑 (default: ".")
```


發布主要版本流程：
* 切換到 develop
* pull 更新到最新
* 建立 release 分支
* 建立 tag
* 推送到服務端

發布次要版本流程：
* 切換到 release
* pull 更新到最新
* 建立 tag
* 推送到服務端
