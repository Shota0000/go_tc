# 作ろうとしているedge環境エミュレートツール概要
- podをエッジサーバとして動かす
- エッジサーバの役割を担うpodには、任意の遅延を入れられるように
- 遅延導入用podを各nodeに配置
![](/image/edge-emulate.png) 
# 利用方法(想定)
1. k8sでedge、クライアント役のpodを作る
- edgeを展開するyamlファイルの一例 
- 上の図で言うとPodA〜PodDを作成
  - 上記の場合、edge-9456bbbf9-7s8nfみたいな名前のpodができる
```
apiVersion: apps/v1
kind: Deployment
metadata: 
  name: edge #任意の名前
  labels:
    app: nginx #これも任意
spec:
  replicas: 4 #podの数
  selector:
    matchLabels:
      app: nginx #ここは下記テンプレート内のラベルと一致させる
  template:
    metadata:
      labels:
        app: nginx
    spec: #pod内にコンテナをさらに追加したい場合は、containers以下に同様の記載を増やしてください
      containers:
      - name: nginx #コンテナ名は任意
        image: nginx:1.14.2 #コンテナのイメージ。自分で用意したものを使ってください。
        ports:
        - containerPort: 80

```
2. 以下のjsonファイルを参考に、設定ファイルを書く
  - その際、podの名前、podのipアドレスが必要なので
    - `kubectl get pods -o=wide` コマンドで調べる
  - podAとpodBがお互いに遅延設定してたら、遅延はそれぞれの合計になることに注意
```
{
    "latency":[
        {
            "from":"edge-9456bbbf9-7s8nf", //遅延を設定するpod名
            "delay":[ 
                {
                    "time":"200ms", //遅延時間設定
                    "to":[
                        "10.40.5.25" //遅延を設定したいipアドレスを配列で指定
                    ]
                },
                {
                    "time":"150ms",
                    "to":[
                        "10.40.6.189" 
                    ]
                }
            ]
        },
        {
            "from":"edge-9456bbbf9-88pfv",
            "delay":[
                {
                    "time":"150ms",
                    "to":[
                        "10.40.5.26" 
                    ]
                }
            ]
        }
    ]
}
```
3. jsonファイルのあるディレクトリで以下のコマンドを撃ち、全nodeからjsonファイルを参照できるようにする
- `kubectl create configmap 任意の名前 --from-file=任意のjsonファイル名`
4. 以下のyamlファイルを用いて、各nodeに遅延導入用podを配置する
- 基本的に、args内、volumesと volumeMounts内のconfigあたり以外触らない
- 図で言うと、pod for setting delay
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
      - image: supercord530/edge-emulate:flute
        imagePullPolicy: Always 
        name: delay 
        args:
          - delay
          - --tc-image
          - supercord530/iproute2:flute #tcコマンドをインストールしているimageを指定。基本このまま
          - set 
          - -f 
          - ../mount/latency.json #以下のmountPathで指定したpath内のjsonを指定。
        volumeMounts:
          - mountPath: /var/run/docker.sock
            name: dockersocket
          - mountPath: /mount #jsonファイルをどこにマウントするか指定
            name: config #volumes内で設定した名前と一致させる
      volumes:
        - name: dockersocket
          hostPath:
            path: /var/run/docker.sock
        - name: config //名前は任意
          configMap: //これを使ってjsonファイルを読み込む
            name: setting-latency # kubectl create configmap 任意の名前 --from-file=任意のjsonファイル名の、任意の名前の部分を指定
```
## もし再度遅延を設定したい場合は、2からやり直す
# メモ
- 遅延導入用podでは、コマンドラインツールが走っている。起動時にargsによりコマンドを指定することで、遅延を設定できる。
- 問題：このやり方だとjsonはどこに配置しよう?
    - jsonではなくk8sのconfigmapを使おう
    - `kubectl create configmap setting-latency --from-file=latency.json`
    - https://qiita.com/oguogura/items/68741b91b70962081504

