# ethutil
An Ethereum util, can transfer eth, check balance, drop pending tx, etc

# Install
```shell
GO111MODULE=on go install github.com/10gic/ethutil
```

# Example
Check balance:
```shell
$ ethutil check-balance --addr 0x756F45E3FA69347A9A973A725E3C98bC4db0b5a0
addr 0x756F45E3FA69347A9A973A725E3C98bC4db0b5a0, balance 0.026990556 ether
```

Transfer eth:
```shell
$ ethutil transfer --private-key 0xXXX --to-addr 0xYYY --amount 1 --unit ether
```

Drop pending tx:
```shell
$ ethutil drop-pending-tx --private-key 0xXXX
```

Dump address from private key:
```shell
$ ethutil dump-address --private-key 0xef065dcbc43081c63c0fbf389ec8df3872d9d61b1bc2e98d7a0a4395d11314d2
private key 0xef065dcbc43081c63c0fbf389ec8df3872d9d61b1bc2e98d7a0a4395d11314d2, addr 0xB2aC853cF815B47903bc19BF4860540306F4f944
```

Generate new private key:
```shell
$ ethutil --terse gen-private-key -n 10
0x4a7a7070d616c70ca7caa5e34dfa944f983d530be4831e6e0086a781a679c601 0x356EC6F0b43bdEB18C291D5e629c1585c3c0BA73
0x692d3eb6ea9df4fb67745b024aa08b6c3f0e14daaba5f13f060fa25ba1d8505a 0x7cdF8bA6cf3599a8892Cc0e7050419d40d03c829
0x9981fcfe901dc14ee20495c2fd61a3895ad1c9bda44d996e392a86d2ecbb5d77 0x8959A4335066876588E3a2362732160cDAb5e1f0
0x06066a29dcd593cab6dfc6c4af6e1a2ccbd08bc768fc03b7d5133fbacff3e705 0x080eEEB739ED46712e560FE9e5cE72879294d33D
0x96590aead1e017fdccc1d60fe43335b445cd9eaab3d1abcaa6019049d5cc60a6 0x68d048a32681CcB9b7D12D9050D18E28eab0d2BF
0x2fa07af249ef7b5412b542107425d5108d2fc1a0a3fee5ddd24e1062fde0564d 0xb1aDee2acEFeE11D0D550DF763Dc7839bCE7Db90
0x2de5588d39947e7d2b35e309aaf63653dad530a85701f76d2db6ee358495b672 0x70d365807bA2946236F8212847DC56C4470E2c87
0x639dbf9774a5e776e1edd0851aabfe0f58857fd2b0d2ecef20ef58dc071df20f 0x959be5AfAa1D5391DE38532f6d0a2b6ae9B761aa
0x8da2192116916e8137d886f24b5fc98ef9a92c4a497ad1a728b8ac89381b307d 0x9355FF732be0e42eEa303C8F22Cd68d50D109549
0x69d5b18f0f6a17b5b0d8b9d5ddc4531d6a14d023b789c4b3d753e40562254a1b 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb
```
