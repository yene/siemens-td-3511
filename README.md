# Siemens TD-3511 reader

Deploy to raspberry pi: `./deploy.sh 192.168.3.a58`

## Notes
Protocol notes

```
/     ?      !    \r    \n
[]byte("/?!\r\n")

ACK    0      5   0    \r    \n
[]byte("\x06060\r\n")
```


## Material and Links
* [Siemens TD-3511 volkszaehler.org](https://wiki.volkszaehler.org/hardware/channels/meters/power/edl-ehz/siemens_td3511)
