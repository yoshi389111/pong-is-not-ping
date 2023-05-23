# Pong-is-not-Ping

This command is pong game.

**PO**ng is **N**ot pin**G**.

Inspired by [kurehajime/pong-command](https://github.com/kurehajime/pong-command)

![](./docs/pong.gif)

## Installation

```
go install github.com/yoshi389111/pong-is-not-ping/cmd/pong@latest
```

## Usage

### Usage:

```
  pong [options] <destination>
```

### Options:

```
Application Options:
  -h, --help                 print help and exit
  -v, --version              print version and exit
  -c, --count=<count>        stop after <count> replies (default: 4)
  -t, --ttl=<ttl>            define time to live (default: 64)
  -p, --padding=<pattern>    contents of padding byte

Arguments:
  <destination>:             dns name or ip address
```

## Related Tech Blog Articles

* dev.to [POng is Not pinG.](https://dev.to/yoshi389111/pong-is-not-ping-323d) English
* qiita [Go言語でpingコマンドっぽい、でもpingコマンドじゃないpongコマンドを作ってみた](https://qiita.com/yoshi389111/items/a289f1c470616c5f10c4) Japanese

## Copyright and License

(C) 2022 SATO, Yoshiyuki

This software is released under the MIT License.
