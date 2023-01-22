# エミュレーション環境をセットアップする方法
分散データストアをk8s環境上にて動作させる例を用いて、セットアップ方法を解説します。
ここでは、flute上に展開する方式を述べます。別に複数サーバーを利用する必要はないので、minicubeやdocker desktopについているk8s環境でも実行できます(動作確認してない。その場合imageが環境依存の可能性あるからbuildしなおす必要あるかも)

# 分散データストアの展開方法
## dockerfileのbuild
分散データストアのイメージを全nodeでbuildする必要があります。docker hubからpullするか、docker registryをlocalに作成すれば、わざわざこんなことはしなくて良いのですが、色々面倒なのでとりあえずこれで。
## headless service,statefulsetを用いたデータストアpodの立ち上げ
以下のyamlファイルを作成してください。その後、`kubectl apply -f ファイル名`でstatefulsetを立ち上げてください。
```
apiVersion: v1
kind: Service
metadata:
  name: edge
  labels:
    app: edge
spec:
  ports:
  - port: 1234 # このportは使いません。設定しないとエラーが出るので指定してます。
  clusterIP: None # headless serviceを作成します。
  selector:
    app: edge #podのラベルと一致させてください
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: c
spec:
  selector:
    matchLabels:
      app: edge # .spec.template.metadata.labelsの値と一致する必要があります
  serviceName: "edge" # 上で設定したserviceの名前と一致させてください.
  replicas: 3 # エッジの数を変えたければここを変えてください。
  template: # ここからpodの設定に入ります。
    metadata:
      labels:
        app: edge # .spec.selector.matchLabelsの値と一致する必要があります
    spec:
      containers:
      - name: db
        image: db:latest
        imagePullPolicy: Never # imageをどこから撮ってくるか指定します.neverだとローカルのみから取得します.指定しなかったらdocker registryから常に取得します。
      - name: mongo
        image: mongo:4.4.18
```
### statefulsetについて
statefulsetとは、statefulなpodを立ち上げるために用いるk8sリソースです。

通常のpodは、同じサービスのpodであればどれにアクセスしても同じ結果を返しますが、statefulsetはそれぞれのpodが個別で状態を持っているため、別々の結果を返します。

statefulsetにより作成されるpodは、statefulsetの名前+番号という名前になります。この例では、edge-0,edge-1,edge2が作成されます。podが作成される順番も、番号の通りになります。edge-0がエラーにより作成されなかった場合、後のpodは作成されません。
### headless serviceについて
この設定ファイルでは、同時にheadless serviceも定義しています。serviceとは、podを外部に公開するためのk8sリソースです。通常のサービスは仮想ipを持ち、そのipにアクセスがあれば、serviceに紐づいたpodにランダムでルーティングします。ここでは、仮想ipを持たないheadless serviceを定義しています。本来のserviceでは、podが個別のドメインを持つことはできませんが、これを用いることでpodはそれぞれ個別のドメインを持つことができます。

ドメイン名は、`$(Pod名).$(Service名).$(namespace名).svc.cluster.local`となります。今回の例では、edge-0のドメイン名は`edge-0.edge.default.svc.cluster.local`となります。namespaceは設定していないためdefaultとなります。

このドメインはk8sクラスタ上の、同一namespace内にあるpodからしか名前解決できません。

## データストアに、mongoのアドレスを教える。
今回、町田さんがnict用に作成したimageを使ってるので、後からmongoのアドレスを教える必要があります。作ったpodに対して、ホストから
`curl -X POST -H "Content-Type: application/json" -d '{"address":"pod名.edge.edge-default.svc.cluster.local","mongo":"localhost"}' podのipアドレス:3000`
を叩いてください。

pod内にあるコンテナ同士はlocalhostでアクセスできます。

podのipアドレスは`kubectl get po -o=wide`で得られます。pod一覧が表示されるのでそこから参照してください。(正直面倒。どうにかならんかな)

- メモ
    - 一応、テキトーなコレクションをpostできたことは確認しました。
        - POST /v1/service/[コレクション名] Body:レコード(JSON形式)を確認した
    - でも、クエリが動くとか詳細は確認してないです...

# 遅延導入ツールのセットアップ
## 設定ファイルの作成
まず、遅延導入用のjsonファイルを作成しましょう。以下に例を書きます。
```
{
    "latency":[
        {
            "from":"edge-0",
            "delay":[
                {
                    "time":"100ms",
                    "to":[
                        "10.40.6.217"
                    ]
                },
                {
                    "time":"150ms",
                    "to":[
                        "10.40.5.79"
                    ]
                }
            ]
        },
        {
            "from":"edge-1",
            "delay":[
                {
                    "time":"100ms",
                    "to":[
                        "10.40.5.78" 
                    ]
                },
                {
                    "time":"200ms",
                    "to":[
                        "10.40.5.79"
                    ]
                }
            ]
        },
        {
            "from":"edge-2",
            "delay":[
                {
                    "time":"150ms",
                    "to":[
                        "10.40.5.78" 
                    ]
                },
                {
                    "time":"200ms",
                    "to":[
                        "10.40.5.217"
                    ]
                }
            ]
        }

    ]
}
```
- `latency`配列に、各podの遅延を設定していきます。設定できる遅延は上りのみであることに注意してください。往復を設定したい場合、例えばedge-0とedge-1の往復に遅延を設定したい場合は、edge-0とedge-1それぞれに遅延を設定する必要があります。
- `from`には、遅延を導入するpod名を指定してください
- `delay`配列には、設定する遅延時間の数だけ配列を設定してください。
- `time`には、遅延時間を設定します。単位まで書いてください。
- `to`配列に、どの相手宛の通信に遅延を導入するかを記述します。現状、podのipアドレスを記述する形にしています(のちにpod名で指定できるようにする。)

設定ファイルが完成したら、`kubectl create configmap 任意の名前 --from-file=作成したjsonファイル名`コマンドを実行してください。configmapとは、設定ファイルを複数のpodから参照できるようにするためのk8sリソースです。

## 遅延導入用podの立ち上げ
以下のyamlファイルを作成してください。その後、`kubectl apply -f ファイル名`で立ち上げてください。
基本的にコメントしてる行以外変えなくていいです。k8sのdeamonsetというリソースを使っています。
```
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: edge-emulate
spec:
  selector:
    matchLabels:
      app: edge-emulate
  template:
    metadata:
      labels:
        app: edge-emulate
        name: edge-emulate
    spec:
      containers:
      - image: supercord530/edge-emulate:flute # 環境によりimageを変更する必要があるかもしれません
        name: delay
        args:
          - delay
          - --tc-image
          - supercord530/iproute2:flute # tcコマンドをインストールしているイメージです。環境によりimageを変更する必要があるかもしれません
          - set 
          - -f 
          - ../mount/latency-ip.json # jsonのファイル名は、先ほど作成した設定ファイルの名前です。
        volumeMounts:
          - mountPath: /var/run/docker.sock
            name: dockersocket
          - mountPath: /mount
            name: config # 下記で設定しているvolume名と一致させてください。
      volumes:
        - name: dockersocket
          hostPath:
            path: /var/run/docker.sock
        - name: config # ここで先ほど作成したconfigmapをマウントします。
          configMap:
            name: edge-delay # 先ほど作成したconfigmapの名前を指定してください。

```
deamonsetについて軽く説明します。podを各nodeに一つ常駐させるためのk8sリソースです。常駐させたpodから遅延を導入しています。
以上で、セットアップは完了です。

# 遅延を変更、リセットする方法
## 変更する方法
`kubectl delete configmap 以前作成したconfigmap名`を実行してください。configmap名を忘れた場合、`kubectl get configmap`で見ることができます。

json設定ファイルを変更し、
`kubectl create configmap 任意の名前 --from-file=作成したjsonファイル名`コマンドを再度実行してください。

その後、`kubectl delete -f 遅延導入用podのyaml`を実行した後に、`kubectl apply -f 遅延導入用podのyaml`を実行してください。jsonファイル名を変更した場合は、yamlファイルの該当する場所を修正した上でapplyしてください。
## リセットする方法
遅延導入用podを削除しただけでは、遅延はリセットされません。
### 方法1
`kubectl delete -f 遅延導入用podのyaml`を実行した後に、yamlファイルのargs部分を以下のように変えてください。
```
args:
          - delay
          - --tc-image
          - supercord530/iproute2:flute
          - --name
          - edge-0 #遅延をresetしたいpodの名前
```
その後、`kubectl apply -f 遅延導入用podのファイル名`を実行してください。
- 現状、一括でのresetに対応してないので、1つずつリセットする必要があります。面倒！
### 方法2
`kubectl delete configmap 以前作成したconfigmap名`を実行してください。configmap名を忘れた場合、`kubectl get configmap`で見ることができます。

json設定ファイルのto部分を空にし、
`kubectl create configmap 任意の名前 --from-file=作成したjsonファイル名`コマンドを再度実行してください。

`kubectl delete -f 遅延導入用podのファイル名`を実行した後に、`kubectl apply -f 遅延導入用podのファイル名`を実行してください。jsonファイル名を変更した場合は、yamlファイルの該当する場所を修正した上でapplyしてください。








