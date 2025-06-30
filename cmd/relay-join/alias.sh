#!/usr/bin/env zsh

# Linux
# ip addr add 127.0.0.2/8 dev lo

 sudo ifconfig lo0 alias 127.0.0.2 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.0.3 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.0.4 netmask 0xff000000

 sudo ifconfig lo0 alias 127.0.1.1 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.1.2 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.1.3 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.1.4 netmask 0xff000000

 sudo ifconfig lo0 alias 127.0.2.1 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.2.2 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.2.3 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.2.4 netmask 0xff000000

 sudo ifconfig lo0 alias 127.0.3.1 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.3.2 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.3.3 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.3.4 netmask 0xff000000

 sudo ifconfig lo0 alias 127.0.4.1 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.4.2 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.4.3 netmask 0xff000000
 sudo ifconfig lo0 alias 127.0.4.4 netmask 0xff000000
