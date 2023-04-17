# A Multiple Network latency Emulation Tool for Kubernetes
## Prerequisites

Minimal required Docker version `v18.06.0`, Kubernetes version `v1.1.0`.

## Usage
### Launch of edge server pod

Create the following YAML file. Then launch pods with `kubectl apply -f 'filename'`.
```
apiVersion: v1
kind: Service
metadata:
  name: edge
  labels:
    app: edge
spec:
  ports:
  - port: 1234 # This port is not used
  clusterIP: None # Create headless service
  selector:
    app: edge # Match pod label
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: edge
spec:
  selector:
    matchLabels:
      app: edge # Must match the value in .spec.template.metadata.labels
  serviceName: "edge" # Match the service name
  replicas: 3 # if you want to change the number of edges, change here.
  template: 
    metadata:
      labels:
        app: edge # Must match the value of .spec.selector.matchLabels
    spec:
      containers:
      - name: ---
        image: --- 
        imagePullPolicy: --- 
```
To create an edge server pod, we use a resource called kubernetes statefulset. This configuration file also uses headless service. This enables each pod to have its domain. The domain name is `$(PodName). $(service name). $(namespace name).svc.cluster.local`.

### Preparation of delay configuration file

Create a JSON file for the delayed deployment tool. An example is written below.

```text
{
    "latency":[
        {
            "from":"edge-0",
            "delay":[
                {
                    "time":"100ms",
                    "to":[
                        "edge-1"
                    ]
                },
                {
                    "time":"150ms",
                    "to":[
                        "edge-2"
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
                        "edge-0" 
                    ]
                },
                {
                    "time":"200ms",
                    "to":[
                        "edge-2"
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
                        "edge-0" 
                    ]
                },
                {
                    "time":"200ms",
                    "to":[
                        "edge-1"
                    ]
                }
            ]
        }

    ]
}
```
- Set the delay for each pod in the `latency` array. Note that only uplink delays can be set. If you want to set up a round trip, for example, a round trip between edge-0 and edge-1, you need to set up a delay for each edge-0 and edge-1.
- The `from` should be the name of the pod where you want to introduce the delay
- In the `delay` array, set as many arrays as the number of delays you want to set.
- In the `time` array, set the delay time. Please write down the unit.
- In the `to` array, specify to which target you want to introduce the delay. 

When the configuration file is complete, run the command `kubectl create configmap 'any name' --from-file= 'created JSON file name'`. configmap is a K8s resource that allows configuration files to be referenced by multiple pods. 

### Launch of delayed implementation tool

Create the following YAML file. Then launch pods with `kubectl apply -f 'filename'`.
You don't have to change anything except the line commenting on. 

```text
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
      - image: supercord530/edge-emulate:flute # You may need to change the image depending on your environment
        name: delay
        args:
          - delay
          - --tc-image
          - supercord530/iproute2:flute # This is an image of the tc command installed. You may need to change the image depending on your environment.
          - set 
          - -f 
          - ../mount/latency-ip JSON # The name of the configuration file you just created.
        volumeMounts:
          - mountPath: /var/run/docker.sock
            name: dockersocket
          - mountPath: /mount
            name: config # Match the volume name set below.
      volumes:
        - name: dockersocket
          hostPath:
            path: /var/run/docker.sock
        - name: config # Mount the configmap you created.
          configMap:
            name: edge-delay # Specify the name of the configmap you created.
```
We are using a K8s resource called deamonset, which is a K8s resource that allows one pod to reside on each node. Delay is introduced from the resident pod.

This completes the setup.

### How to change or reset delays

#### How to change the delay

Run `kubectl delete configmap 'previously created configmap name'`. If you forget the configmap name, you can see it with `kubectl get configmap`.

Modify the JSON configuration file, Run 
`kubectl create configmap 'any name' --from-file='created JSON file name'` command again.

If you have changed the JSON file name, modify the corresponding location in the YAML file, and then apply.

#### How to reset the delay

Deleting the delay implementation pod does not reset the delay.

##### Method 1

Run `kubectl delete configmap 'previously created configmap name'`. If you forget the configmap name, you can see it with `kubectl get configmap`.

After emptying the `to` part of the JSON configuration file,
Execute `kubectl create configmap 'any name' --from-file= JSON file name'` command again.

If you have changed the JSON file name, modify the corresponding location in the YAML file, and then run `kubectl delete -f YAML file name`. If you have changed the JSON file name, modify the corresponding location in the YAML file and then apply.

##### Method 2

After running `kubectl delete -f 'YAML for delayed installation pod'`, change the args part of the YAML file as follows.
```
args:
   - delay
   - --tc-image
   - supercord530/iproute2:flute
   - --name
   - edge-0
```
After that, run `kubectl apply -f 'filename of pod for delay installation'`.

## License

Code is under the [Apache License v2](https://www.apache.org/licenses/LICENSE-2.0.txt).