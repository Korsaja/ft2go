This program is linked to the C flow-tools api for faster statistics collection.<br>For it to work, you need to
 install:<br>
`sudo apt-get update`<br>
`sudo apt-get install flow-tools-dev`<br>
There is support for sending the report by email **csv** file format.<br>
Example config file:<br>
```toml
[smtp]
server = ""
port = 0
mail = ""
pass = ""
[[devices]]
[[devices.attr]]
ip = "192.168.10.1"
name = "router1"
[[devices.attr]]
ip = "192.168.20.1"
name = "router2"
# filters
[[nfilters]]
[[nfilters.attr]]
ips = ["192.168.20.0/24"]
name = "vlan20"
iface = 1
[[nfilters]]
[[nfilters.attr]]
ips = ["192.168.20.0/24","192.168.10.0/24"]
name = "allvlan"
iface = 2
```
If you need to add netflow fields see names and data types add them to the `generator.go`:<br>
https://github.com/adsr/flow-tools/blob/master/lib/ftlib.h#L613
