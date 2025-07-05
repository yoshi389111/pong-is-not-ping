# PONG(6)

## NAME

`pong` - **PO**ng is **N**ot pin**G**; simulate ping as an interactive Pong match

## SYNOPSIS

```txt
pong [OPTIONS] <DESTINATION>
```

## DESCRIPTION

`pong` is a terminal-based game that parodies the classic ping utility. Instead of simply sending ICMP ECHO requests and waiting for replies, you play a Pong-style game with them.

When you run `pong` with a target host specified as `<DESTINATION>`, your local host (displayed on the left edge of the screen) will "send" ICMP ECHO packets as if they were Pong balls. The user controls the paddle on the right side (representing the target host) and must return these packets back to the local host.

Between them is the Gateway — a CPU-controlled paddle that always bounces the ICMP ECHO packets back toward the target host. The gateway doesn't understand that it should forward packets; it just reflects them blindly, reducing the TTL each time. You need to avoid the gateway’s deflections and ensure your packets make it back to the local host before their TTL expires (starting at 64 by default, changeable with `-t`).

The longer you rally the packets back and forth, the faster they travel, increasing the difficulty.

By default, the local host sends 4 ICMP ECHO packets (changeable with `-c`). When the last packet is returned or lost, or when the user quits the game, pong will display ping-style statistics summarizing the round-trip success and loss.

It's ping meets pong — test your reflexes while staying true to the spirit of the network diagnostic classic.

## OPTIONS

- `-h`, `--help`
  - Print help and exit.

- `-v`, `--version`
  - Print version and exit.

- `-c`, `--count=<count>`
  - Stop after `<count>` replies. The default is 4.

- `-t`, `--ttl=<ttl>`
  - Define the time to live (TTL) for the packets. The default is 64.

- `-p`, `--padding=<pattern>`
  - Specify the padding string for ICMP echo packets.

- `<DESTINATION>`
  - Specify the target host by IP address or hostname.

## EXAMPLE

```shell
pong 192.168.1.1
```

![pong](./docs/pong.gif)

```console
$ pong 192.168.1.1
PONG 192.168.1.1(192.168.1.1) 9 bytes of data.
9 bytes from 192.168.1.1: icmp_seq=1 ttl=51 time=63 sec
9 bytes from 192.168.1.1: icmp_seq=2 ttl=63 time=16 sec
9 bytes from 192.168.1.1: icmp_seq=3 ttl=62 time=21 sec
9 bytes from 192.168.1.1: icmp_seq=4 ttl=61 time=25 sec
--- 192.168.1.1 pong statistics ---
4 packets transmitted, 4 packets received, 0% packet loss, time 133 sec
```

## INSTALLATION

```shell
go install github.com/yoshi389111/pong-is-not-ping/cmd/pong@latest
```

## SEE ALSO

- Inspired by [kurehajime/pong-command](https://github.com/kurehajime/pong-command)
- [POng is Not pinG. (dev.to)](https://dev.to/yoshi389111/pong-is-not-ping-323d) ― English article explaining this project
- [Go言語でpingコマンドっぽい、でもpingコマンドじゃないpongコマンドを作ってみた (Qiita)](https://qiita.com/yoshi389111/items/a289f1c470616c5f10c4) ― Japanese article explaining this project

## LICENSE

This software is released under the MIT License.

&copy; 2022 SATO, Yoshiyuki
