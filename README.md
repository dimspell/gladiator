# Gladiator

This repository is a monorepo that contains multiple projects that helps to play a Dispel Multiplayer game.

* **console** - A main server handling the state preservation and communication with the clients.
* **backend** - A backend server that handles the commands send by the `DispelMulti.exe`. It exchanges the data with the **console**.
* **launcher** - A launcher for the game. It is a GUI app that is responsible for resolving all problems with interconnecting the game, the **console** and the **backend**.

## Usage

1. After **Dispel Colosseum** installation, open `regedit.exe` to find `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi` and replace `Server` key with `localhost` as a new value.

```diff
- HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi\Server dispel.e2soft.com
+ HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi\Server localhost
```

## Troubleshooting

### Windows 

#### Cannot start the backend server

> "An attempt was made to access a socket in a way forbidden by its access permissions."

Restart of the Host Network Service on Windows might fix this error. Open an elevated (with admin permissions) Command Prompt and run:

```console
net stop hns
net start hns
```

### Linux  

#### Cannot bind the 127.0.0.X for the relayed hosts

```bash
ip addr add 127.0.0.2/8 dev lo
```

#### QUIC error about the buffer being too small

In the backend logs, you might notice the following fragment:

> ... See https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes for details

On Linux you need to extend the memory for the buffer. 
Note that these settings are not persisted across reboots.

```bash
sysctl -w net.core.rmem_max=7500000
sysctl -w net.core.wmem_max=7500000
```


### MacOS

#### Cannot bind the 127.0.0.X for the relayed hosts

Note that these settings are not persisted across reboots.

```bash
sudo ifconfig lo0 alias 127.0.0.2 netmask 0xff000000
sudo ifconfig lo0 alias 127.0.0.3 netmask 0xff000000
sudo ifconfig lo0 alias 127.0.0.4 netmask 0xff000000
# ...
```
