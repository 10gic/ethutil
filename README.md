# ethutil
An Ethereum util, can transfer eth, check balance, drop pending tx, dump address from private key or mnemonic, call any contract function etc

# Install
```shell
GO111MODULE=on go install github.com/10gic/ethutil@latest
```

# Usage
```txt
An Ethereum util, can transfer eth, check balance, drop pending tx, etc

Usage:
  ethutil [command]

Available Commands:
  balance               Check eth balance for address
  transfer              Transfer amount of eth to target-address
  call                  Invokes the (paid) contract method
  query                 Invokes the (constant) contract method
  deploy                Deploy contract
  drop-tx               Drop pending tx for address
  gen-private-key       Generate eth private key and its address
  dump-address          Dump address from private key or mnemonic
  compute-contract-addr Compute contract address before deployment
  decode-tx             Decode raw transaction
  help                  Help about any command

Flags:
      --dry-run              do not broadcast tx
      --gas-limit uint       the gas limit
      --gas-price string     the gas price, unit is gwei.
  -h, --help                 help for ethutil
      --node string          mainnet | ropsten | kovan | rinkeby | sokol, the node type (default "kovan")
      --node-url string      the target connection node url, if this option specified, the --node option is ignored
  -k, --private-key string   the private key, eth would be send from this account
      --show-raw-tx          print raw signed tx
      --terse                produce terse output

Use "ethutil [command] --help" for more information about a command.
```

# Example
## Check Balance
Check balance of an address:
```shell
$ ethutil balance 0x756F45E3FA69347A9A973A725E3C98bC4db0b5a0
addr 0x756F45E3FA69347A9A973A725E3C98bC4db0b5a0, balance 0.026990556 ether
```

Check balance (terse output, multiple addresses):
```shell
$ ethutil --terse balance 0x756F45E3FA69347A9A973A725E3C98bC4db0b5a0 0xf42f905231c770f0a406f2b768877fb49eee0f21
0x756F45E3FA69347A9A973A725E3C98bC4db0b5a0 0.026990556
0xf42f905231c770f0a406f2b768877fb49eee0f21 408.33061699
```

## Transfer ETH
Transfer 1 ETH to 0xB2aC853cF815B47903bc19BF4860540306F4f944:
```shell
$ ethutil transfer 0xB2aC853cF815B47903bc19BF4860540306F4f944 1 --private-key 0xXXXX
```

## Contract Interaction
Invokes the (paid) contract method:
```shell
$ ethutil --node mainnet --private-key 0xXXXX call 0xdac17f958d2ee523a2206206994597c13d831ec7 'transfer(address, uint256)' 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
```

Invokes the (paid) contract method with abi file:
```shell
$ ethutil --node mainnet --private-key 0xXXXX call 0xdac17f958d2ee523a2206206994597c13d831ec7 --abi-file path/to/abi transfer 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
```

Invokes the (constant) contract method:
```shell
$ ethutil --node mainnet query 0xdac17f958d2ee523a2206206994597c13d831ec7 'balanceOf(address) returns (uint256)' 0x703662e526d2b71944fbfb9d87f61de3e0f0f290
ret0 = 1100000000000
```

Invokes the (constant) contract method with abi file:
```shell
$ ethutil --node mainnet query 0xdac17f958d2ee523a2206206994597c13d831ec7 --abi-file path/to/abi balanceOf 0x703662e526d2b71944fbfb9d87f61de3e0f0f290
```

## Deploy Contract
Deploy a contract:
```shell
$ ethutil --private-key 0xXXXX deploy --bin-file contract-bytecode.bin
```

## Drop Pending Tx
```shell
$ ethutil drop-tx --private-key 0xXXXX
```

## Generate New Private Key
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

## Dump Address From Private Key
```shell
$ ethutil dump-address --private-key-or-mnemonic 0xef065dcbc43081c63c0fbf389ec8df3872d9d61b1bc2e98d7a0a4395d11314d2
private key 0xef065dcbc43081c63c0fbf389ec8df3872d9d61b1bc2e98d7a0a4395d11314d2, addr 0xB2aC853cF815B47903bc19BF4860540306F4f944
```

## Compute Contract Address
Compute contract address before deployment:
```shell
$ ethutil compute-contract-addr --deployer-addr 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb --nonce 0
deployer address 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb
nonce 0
contract address 0x3bb8C061Ec6EdB3E78777b983b96468CC4799888 
```

Compute contract address (CREATE2) before deployment:
```shell
$ ethutil compute-contract-addr --deployer-addr 0x0000000000000000000000000000000000000000 --salt 0x0000000000000000000000000000000000000000000000000000000000000000 --init-code 0x00
deployer address 0x0000000000000000000000000000000000000000
salt 0x0000000000000000000000000000000000000000000000000000000000000000
init code 0x00
contract address 0x4D1A2e2bB4F88F0250f26Ffff098B0b30B26BF38
```

## Decode Raw Transaction
```shell
$ ethutil decode-raw-tx --hex-data 0xf86c808504e3b2920082520894428cf082d321d435ff0e1f8a994e01f976f19c118809b5552f5abade008026a00a27decf27241dca4e5d82bd5b7c1fbcc3f09c35a2a05cb967f2983d148ad6aba0596e9baa40ab157f5b1b0d66746472550ba9000d4154e3faa43ccce00b030452
basic info (see eip155):
nonce = 0
gas price = 21000000000, i.e. 21 Gwei
gas limit = 21000
to = 0x428Cf082D321d435fF0e1F8a994e01f976F19c11
value = 699558979000000000, i.e. 0.699558979 Ether
input data (hex) = 
chain id = 1
v = 38
r (hex) = a27decf27241dca4e5d82bd5b7c1fbcc3f09c35a2a05cb967f2983d148ad6ab
s (hex) = 596e9baa40ab157f5b1b0d66746472550ba9000d4154e3faa43ccce00b030452

derived info:
txid (hex) = a8208564aa36d095973ce979df5bda03568ae0fb55f76517f1d91438bba84390
hash before ecdsa sign (hex) = 75fee2d3e846aacfcd167febf6af8d17b6fb73188a06bcf7cb5b626a347bad54
ecdsa recovery id = 1
uncompressed 65 bytes public key of sender (hex) = 04a68b04c516ef2bec4e598c825cac73350012b2fe6f798270be34f800bab44024dc3580906d2f709013745a79f37cc4e3fd651289a46f5bdafd6e9da63389aec8
address of sender = 0xf7033D6010E8F2E12b810883e1c28CAcd6D25B16
```
