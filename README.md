# Edge-emulate: emulate edge environment latency
- 現状，tcコマンドとgoがインストールされていないと動かない
    - そこはpumbaと同じやり方で対応するつもり
        - pumbaのやり方
        - goの問題
            - pumbaのpodでgoを動かし，tcコマンドだけを各podに送信
        - tcコマンドをインストールしないと問題
            - `--tc-image gaiadocker/iproute2` とかいうオプションつけて対応してた．詳細はまだみてない...

# 利用法
```
$ delay help

NAME:
   edge-emulate delay - if use set reset,init or add

USAGE:
   edge-emulate delay [global options] command [command options] [arguments...]

VERSION:
   0.1.0

COMMANDS:
   reset  reset delay
   init   if use set -t，ーi
   add    if use set -t，ーi

GLOBAL OPTIONS:
   --help, -h  show help
```
## 遅延の初期設定
- 例：`delay init -t 100ms -i 192.168.11.10 -i 192.168.11.20 -i ...`
    - -t オプションは遅延を，-i オプションはipアドレスを指定する．ipアドレスは複数指定可能
```
NAME:
   edge-emulate delay init - if use set -t，ーi

USAGE:
   edge-emulate delay init [command options] [arguments...]

OPTIONS:
   -t value, --time value  Decide how much to delay
   -i value, --ip value    Specify ip address for delay
```
## 遅延のリセット
- 例：`delay reset`
```
NAME:
   edge-emulate delay reset - reset delay

USAGE:
   edge-emulate delay reset
```

## 遅延の追加
- 例：`delay add -t 100ms -p 5 -i 192.168.11.10 -i 192.168.11.20 -i ...`
    - -t，-iは初期設定と同様．
    - -pは優先順位．現状，ipアドレスに紐づく遅延を後から変更することができない．そこで，新たに優先順位の高いルールを追加することによって，擬似的な変更を実現する．
    デフォルトの優先順位は100 (番号が若いほど優先順位が高い)
```
NAME:
   edge-emulate delay add - if use set -t，ーi

USAGE:
   edge-emulate delay add [command options] [arguments...]

OPTIONS:
   -t value, --time value      Decide how much to delay
   -i value, --ip value        Specifies an ip address. Multiple addresses are possible.
   -p value, --priority value  Specify priority as an integer
```