tc qdisc del dev eth0 root
tc qdisc add dev eth0 root handle 1: htb default 10
tc class add dev eth0 parent 1: classid 1:1 htb rate 100Gbit 

echo "create class"
for i in `seq 1 100`
do
 tc class add dev eth0 parent 1:1 classid 1:${i}0 htb rate 10Gbit
done

tc qdisc add dev eth0 parent 1:10 handle 101: pfifo limit 1000

echo "add qdisc to class"
for i in `seq 2 100`
do
 tc qdisc add dev eth0 parent 1:${i}0 handle 1${i}: netem delay 100ms
done

echo "add filter"
for i in `seq 1 100`
do
 tc filter add dev eth0 protocol ip parent 1: prio 1 u32 match ip dst 10.40.1.${i}/32 flowid 1:${i}0
done


