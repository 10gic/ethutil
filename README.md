# ethutil
An Ethereum util, can transfer eth, check balance, call any contract function etc. All EVM-compatible chains are supported.

# Documentation
```txt
An Ethereum util, can transfer eth, check balance, call any contract function etc. All EVM-compatible chains are supported.

Usage:
  ethutil [command]

Available Commands:
  balance                 Check eth balance for address
  transfer                Transfer native token
  call                    Invoke the (paid) contract method
  query                   Invoke the (constant) contract method
  deploy                  Deploy contract
  deploy-erc20            Deploy an ERC20 token
  drop-tx                 Drop pending tx for address
  4byte                   Get the function signatures for the given selector from https://openchain.xyz/signatures or https://www.4byte.directory
  encode-param            Encode input arguments, it's useful when you call contract's method manually
  gen-key                 Generate eth mnemonic words, private key, and its address
  dump-address            Dump address from mnemonics or private key or public key
  compute-contract-addr   Compute contract address before deployment
  build-raw-tx            Build raw transaction, the output can be used by rpc eth_sendRawTransaction
  broadcast-tx            Broadcast tx by rpc eth_sendRawTransaction
  decode-tx               Decode raw transaction
  code                    Get runtime bytecode of a contract on the blockchain, or EIP-7702 EOA code.
  erc20                   Call ERC20 contract, a helper for subcommand call/query
  keccak                  Compute keccak hash of data. If data is a existing file, compute the hash of the file content
  personal-sign           Create EIP191 personal sign
  eip712-sign             Create EIP712 sign
  aa-simple-account       AA (EIP4337) simple account, owned by an EOA account
  download-src            Download source code of contract from block explorer platform, eg. etherscan.
  eip7702-set-eoa-code    Set EOA account code, see EIP-7702. Just use 0x0000000000000000000000000000000000000000 when you want to clear the code.
  eip7702-sign-auth-tuple Sign EIP-7702 authorization tuple, see EIP-7702.
  help                    Help about any command
  completion              Generate the autocompletion script for the specified shell

Flags:
      --chain string                      mainnet | sepolia | sokol | bsc. This parameter can be set as the chain ID, in this case the rpc comes from https://chainid.network/chains_mini.json (default "sepolia")
      --dry-run                           do not broadcast tx
      --gas-limit uint                    the gas limit
      --gas-price string                  the gas price, unit is gwei.
  -h, --help                              help for ethutil
      --max-fee-per-gas string            maximum fee per gas they are willing to pay total, unit is gwei. see eip1559
      --max-priority-fee-per-gas string   maximum fee per gas they are willing to give to miners, unit is gwei. see eip1559
      --node-url string                   the target connection node url, if this option specified, the --chain option is ignored
      --nonce int                         the nonce, -1 means check online (default -1)
  -k, --private-key string                the private key, eth would be send from this account
      --show-estimate-gas                 print estimate gas of tx
      --show-input-data                   print input data of tx
      --show-pre-hash                     print pre hash, the input of ecdsa sign
      --show-raw-tx                       print raw signed tx
      --terse                             produce terse output
      --tx-type string                    eip155 | eip1559, the type of tx your want to send (default "eip1559")

Use "ethutil [command] --help" for more information about a command.
```

# Install
```shell
go install github.com/10gic/ethutil@latest
```

# Usage Example
## Check Balance (extremely fast for multiple addresses)
Check balance of an address:
```shell
$ ethutil --chain mainnet balance 0x79047aBf3af2a1061B108D71d6dc7BdB06474790
addr 0x79047aBf3af2a1061B108D71d6dc7BdB06474790, balance 231.905355677037965414 ether
```

Check balances of multiple addresses, it's really fast (only take about 10s for 10000 addresses):
```shell
$ ethutil --chain mainnet balance --input-file address.txt         # address.txt format: one address per line
addr 0x00000000219ab540356cbb839cbe05303d7705fa, balance 1.989730000000000005 ether
addr 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2, balance 33.749145122485331533 ether
addr 0xbe0eb53f46cd790cd13851d5eff43d12404d33e8, balance 1 ether
addr 0x8315177ab297ba92a06054ce80a67ed4dbd7ed3a, balance 0 ether
......
```

## Transfer ETH
Transfer 1 ETH to 0xB2aC853cF815B47903bc19BF4860540306F4f944:
```shell
$ ethutil --chain mainnet transfer 0xB2aC853cF815B47903bc19BF4860540306F4f944 1 --private-key 0xXXXX
```

Transfer all remaining ETH to 0xB2aC853cF815B47903bc19BF4860540306F4f944:
```shell
$ ethutil --chain mainnet transfer 0xB2aC853cF815B47903bc19BF4860540306F4f944 all --private-key 0xXXXX
```

## Contract Interaction
Invokes the (paid) contract method:
```shell
$ ethutil --chain mainnet --private-key 0xXXXX call 0xdac17f958d2ee523a2206206994597c13d831ec7 'transfer(address, uint256)' 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
```

Invokes the (paid) contract method with abi file:
```shell
$ ethutil --chain mainnet --private-key 0xXXXX call 0xdac17f958d2ee523a2206206994597c13d831ec7 --abi-file path/to/abi transfer 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
```

Invokes the (constant) contract method:
```shell
$ ethutil --chain mainnet query 0xdac17f958d2ee523a2206206994597c13d831ec7 'balanceOf(address) returns (uint256)' 0x703662e526d2b71944fbfb9d87f61de3e0f0f290
ret0 = 1100000000000
```

Invokes the (constant) contract method with abi file:
```shell
$ ethutil --chain mainnet query 0xdac17f958d2ee523a2206206994597c13d831ec7 --abi-file path/to/abi balanceOf 0x703662e526d2b71944fbfb9d87f61de3e0f0f290
```

## Deploy Contract
Deploy a contract:
```shell
$ ethutil --private-key 0xXXXX deploy --bin-file Contract1_sol_Contract1.bin
```

The binary file Contract1_sol_Contract1.bin can be generated by `solcjs`, for example:
```
$ solcjs --bin Contract1.sol      # generate Contract1_sol_Contract1.bin
```

## Deploy A ERC20 Token
Deploy A ERC20 Token (use default setting: totalSupply = "10000000000000000000000000", name = "A Simple ERC20", symbol = "TEST", decimals = 18)
```shell
$ ethutil deploy-erc20
```

## Drop Pending Tx
```shell
$ ethutil drop-tx --private-key 0xXXXX
```

## Encode Param
An example:
```shell
$ ethutil encode-param 'transfer(address, uint256)' 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
MethodID: 0xa9059cbb
[0]:  0x0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb
[1]:  0x00000000000000000000000000000000000000000000000000000000000f4240
encoded parameters (input data) = 0xa9059cbb0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb00000000000000000000000000000000000000000000000000000000000f4240
```

Another example:
```shell
$ ethutil encode-param 'swapExactETHForTokens(uint256 amountOutMin, address[] path, address to, uint256 deadline)' 12939945098273591402279 '[0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2, 0x9Ed8e7C9604790F7Ec589F99b94361d8AAB64E5E]' 0x95206727FA3DD2FA32cd0BfE1fd40736B525CF11 1615952806
MethodID: 0x7ff36ab5
[0]:  0x0000000000000000000000000000000000000000000002bd79cff41cc68c1f27
[1]:  0x0000000000000000000000000000000000000000000000000000000000000080
[2]:  0x00000000000000000000000095206727fa3dd2fa32cd0bfe1fd40736b525cf11
[3]:  0x0000000000000000000000000000000000000000000000000000000060517ba6
[4]:  0x0000000000000000000000000000000000000000000000000000000000000002
[5]:  0x000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
[6]:  0x0000000000000000000000009ed8e7c9604790f7ec589f99b94361d8aab64e5e
encoded parameters (input data) = 0x7ff36ab50000000000000000000000000000000000000000000002bd79cff41cc68c1f27000000000000000000000000000000000000000000000000000000000000008000000000000000000000000095206727fa3dd2fa32cd0bfe1fd40736b525cf110000000000000000000000000000000000000000000000000000000060517ba60000000000000000000000000000000000000000000000000000000000000002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc20000000000000000000000009ed8e7c9604790f7ec589f99b94361d8aab64e5e
```

Only encode arguments (without function selector):
```shell
$ ethutil encode-param '(address, uint256)' 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
[0]:  0x0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb
[1]:  0x00000000000000000000000000000000000000000000000000000000000f4240
encoded parameters (input data) = 0x0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb00000000000000000000000000000000000000000000000000000000000f4240
```

## Generate New Private Key
Generate mnemonic words and private key:
```shell
$ ethutil gen-key
mnemonic: obvious element orbit option muffin crop abuse duck general mule satoshi doll
private key: 0x836263588c9ea3ffa2a73b71a32d4eb886779d1e0e25f6324c582d2f1008d57f
public key: 0x04049817a72deed750a27a7abc772169378f1547133861d564eba86561a45f861658bcf9ba9cd1be852df8e1c2df76492402c665b9644fb2a37b29644ac17c0d45
addr: 0x2Ed852F7F064E56aa60fDA0a703ed4A7DCC5F9fb
```

Generate multiple keys:
```shell
$ ethutil --terse gen-key -n 10
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

## Dump Address
Dump address from mnemonic:
```shell
$ ethutil dump-address 'obvious element orbit option muffin crop abuse duck general mule satoshi doll'
private key: 0x836263588c9ea3ffa2a73b71a32d4eb886779d1e0e25f6324c582d2f1008d57f
public key: 0x04049817a72deed750a27a7abc772169378f1547133861d564eba86561a45f861658bcf9ba9cd1be852df8e1c2df76492402c665b9644fb2a37b29644ac17c0d45
addr: 0x2Ed852F7F064E56aa60fDA0a703ed4A7DCC5F9fb
```

Dump address from mnemonic with derivation path:
```shell
$ ethutil dump-address --derivation-path "m/44'/61'/0'/0/0" 'obvious element orbit option muffin crop abuse duck general mule satoshi doll'
private key: 0x25b2cb153e31292c0ad8f394292d0f3838338763a97c23376d4ff4d30901b487
public key: 0x044c296796ec2bc3c2f02b77064d71bd198254d3c8f72a648a4f3f59e0830b20e53299dad3a5edccc8d73e9821d92e508b108a3e8c1e689a3b5914260e33dfe278
addr: 0xeBe604aD190F42F587b01308Ce084dF6163A0411
```

Dump address from private key:
```shell
$ ethutil dump-address 0x836263588c9ea3ffa2a73b71a32d4eb886779d1e0e25f6324c582d2f1008d57f
private key: 0x836263588c9ea3ffa2a73b71a32d4eb886779d1e0e25f6324c582d2f1008d57f
public key: 0x04049817a72deed750a27a7abc772169378f1547133861d564eba86561a45f861658bcf9ba9cd1be852df8e1c2df76492402c665b9644fb2a37b29644ac17c0d45
addr: 0x2Ed852F7F064E56aa60fDA0a703ed4A7DCC5F9fb
```

Dump address from public key:
```shell
$ ethutil dump-address 0x04049817a72deed750a27a7abc772169378f1547133861d564eba86561a45f861658bcf9ba9cd1be852df8e1c2df76492402c665b9644fb2a37b29644ac17c0d45
public key: 0x04049817a72deed750a27a7abc772169378f1547133861d564eba86561a45f861658bcf9ba9cd1be852df8e1c2df76492402c665b9644fb2a37b29644ac17c0d45
addr: 0x2Ed852F7F064E56aa60fDA0a703ed4A7DCC5F9fb
```

## Compute Contract Address
Compute contract address before deployment:
```shell
$ ethutil compute-contract-addr 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb --nonce 0
deployer address 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb
nonce 0
contract address 0x3bb8C061Ec6EdB3E78777b983b96468CC4799888 
```

Compute contract address (CREATE2) before deployment:
```shell
$ ethutil compute-contract-addr 0x0000000000000000000000000000000000000000 --salt 0x0000000000000000000000000000000000000000000000000000000000000000 --init-code 0x00
deployer address 0x0000000000000000000000000000000000000000
salt 0x0000000000000000000000000000000000000000000000000000000000000000
init code 0x00
contract address 0x4D1A2e2bB4F88F0250f26Ffff098B0b30B26BF38
```

## Build Raw Transaction
```shell
$ ethutil build-raw-tx 0x356EC6F0b43bdEB18C291D5e629c1585c3c0BA73 0x7cdF8bA6cf3599a8892Cc0e7050419d40d03c829 1 --private-key 0x4a7a7070d616c70ca7caa5e34dfa944f983d530be4831e6e0086a781a679c601
signed raw tx (can be used by rpc eth_sendRawTransaction) = 0x02f866058082076f820778825208947cdf8ba6cf3599a8892cc0e7050419d40d03c8290180c001a0d2f1549d9d16b2cdf9d617011dbfc2a9394dccd21bff307c89408191c55ae811a07bf86a7a65beb324ddd0cdb5c7303d2f84b3003b8e918234173982e34f13eff7
```

## Broadcast Transaction
```shell
$ ethutil broadcast-tx 0x02f866058082076f820778825208947cdf8ba6cf3599a8892cc0e7050419d40d03c8290180c001a0d2f1549d9d16b2cdf9d617011dbfc2a9394dccd21bff307c89408191c55ae811a07bf86a7a65beb324ddd0cdb5c7303d2f84b3003b8e918234173982e34f13eff7
```

## Decode Raw Transaction
```shell
$ ethutil decode-tx 0xf86c808504e3b2920082520894428cf082d321d435ff0e1f8a994e01f976f19c118809b5552f5abade008026a00a27decf27241dca4e5d82bd5b7c1fbcc3f09c35a2a05cb967f2983d148ad6aba0596e9baa40ab157f5b1b0d66746472550ba9000d4154e3faa43ccce00b030452
basic info:
type = eip155, i.e. legacy transaction
chainId = 1 (0x01)
nonce = 0 (0x0)
gasPrice = 21000000000 (0x04e3b29200), i.e. 21 Gwei
gasLimit = 21000 (0x5208)
to = 0x428Cf082D321d435fF0e1F8a994e01f976F19c11
value = 699558979000000000 (0x09b5552f5abade00), i.e. 0.699558979 Ether
data (hex) = 
v = 38 (0x26)
r (hex) = 0a27decf27241dca4e5d82bd5b7c1fbcc3f09c35a2a05cb967f2983d148ad6ab
s (hex) = 596e9baa40ab157f5b1b0d66746472550ba9000d4154e3faa43ccce00b030452

derived info:
txid (hex) = a8208564aa36d095973ce979df5bda03568ae0fb55f76517f1d91438bba84390
hash before ecdsa sign (hex) = 75fee2d3e846aacfcd167febf6af8d17b6fb73188a06bcf7cb5b626a347bad54
ecdsa recovery id = 1
uncompressed public key of sender (hex) = 04a68b04c516ef2bec4e598c825cac73350012b2fe6f798270be34f800bab44024dc3580906d2f709013745a79f37cc4e3fd651289a46f5bdafd6e9da63389aec8
sender = 0xf7033D6010E8F2E12b810883e1c28CAcd6D25B16
```

## Get Contract Runtime Bytecode
```shell
$ ethutil --chain mainnet code 0xd152f549545093347a162dce210e7293f1452150
runtime bytecode of contract 0xd152f549545093347a162dce210e7293f1452150 is 0x608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806351ba162c1461005c578063c73a2d60146100cf578063e63d38ed14610142575b600080fd5b34801561006857600080fd5b506100cd600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001908201803590602001919091929391929390803590602001908201803590602001919091929391929390505050610188565b005b3480156100db57600080fd5b50610140600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001908201803590602001919091929391929390803590602001908201803590602001919091929391929390505050610309565b005b6101866004803603810190808035906020019082018035906020019190919293919293908035906020019082018035906020019190919293919293905050506105b0565b005b60008090505b84849050811015610301578573ffffffffffffffffffffffffffffffffffffffff166323b872dd3387878581811015156101c457fe5b9050602002013573ffffffffffffffffffffffffffffffffffffffff1686868681811015156101ef57fe5b905060200201356040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050602060405180830381600087803b1580156102ae57600080fd5b505af11580156102c2573d6000803e3d6000fd5b505050506040513d60208110156102d857600080fd5b810190808051906020019092919050505015156102f457600080fd5b808060010191505061018e565b505050505050565b60008060009150600090505b8585905081101561034657838382818110151561032e57fe5b90506020020135820191508080600101915050610315565b8673ffffffffffffffffffffffffffffffffffffffff166323b872dd3330856040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050602060405180830381600087803b15801561041d57600080fd5b505af1158015610431573d6000803e3d6000fd5b505050506040513d602081101561044757600080fd5b8101908080519060200190929190505050151561046357600080fd5b600090505b858590508110156105a7578673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb878784818110151561049d57fe5b9050602002013573ffffffffffffffffffffffffffffffffffffffff1686868581811015156104c857fe5b905060200201356040518363ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b15801561055457600080fd5b505af1158015610568573d6000803e3d6000fd5b505050506040513d602081101561057e57600080fd5b8101908080519060200190929190505050151561059a57600080fd5b8080600101915050610468565b50505050505050565b600080600091505b858590508210156106555785858381811015156105d157fe5b9050602002013573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc858585818110151561061557fe5b905060200201359081150290604051600060405180830381858888f19350505050158015610647573d6000803e3d6000fd5b5081806001019250506105b8565b3073ffffffffffffffffffffffffffffffffffffffff1631905060008111156106c0573373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f193505050501580156106be573d6000803e3d6000fd5b505b5050505050505600a165627a7a72305820104eaf57909eb0d29f37ba9e3196e8e88438f83546136cf61270ca5d3b491e160029
```

## ERC20 Interaction
The subcommand `erc20` is a helper for subcommand `call/query`.

Example of check ERC20 balance:
```shell
$ ethutil --chain mainnet erc20 0xdac17f958d2ee523a2206206994597c13d831ec7 balanceOf 0x703662e526d2b71944fbfb9d87f61de3e0f0f290

```

Example of transfer ERC20:
```shell
$ ethutil --chain mainnet --private-key 0xXXXX erc20 0xdac17f958d2ee523a2206206994597c13d831ec7 transfer 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb 1000000
```

## Compute keccak hash
```shell
$ ethutil keccak hello
1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8
$ ethutil keccak --hex 0xabcdef
800d501693feda2226878e1ec7869eef8919dbc5bd10c2bcd031b94d73492860
```

## Download source of verified contract
```shell
$ ethutil --chain mainnet download-src 0xdac17f958d2ee523a2206206994597c13d831ec7 -d output
2021/12/12 21:25:44 Current chain is mainnet
2021/12/12 21:25:45 saving output/TetherToken.sol
```

## Set EOA code (EIP-7702)
```shell
$ ethutil eip7702-set-eoa-code 0x2Ed852F7F064E56aa60fDA0a703ed4A7DCC5F9fb --private-key 0xXXXX # set code for EOA
$ ethutil eip7702-set-eoa-code 0x0000000000000000000000000000000000000000 --private-key 0xXXXX # clear code for EOA
```

## Sign EIP-7702 authorization tuple
```shell
$ ethutil eip7702-sign-auth-tuple 17000 0x2Ed852F7F064E56aa60fDA0a703ed4A7DCC5F9fb 1 --private-key 0xXXXX # Sign <chain-id> <delegate-to> <nonce> 
```

# Known Issue
## daily request count exceeded, request rate limited
If `panic: daily request count exceeded, request rate limited` appears, please use your own node url. It can be changed by option `--node-url`, for example `--node-url wss://mainnet.infura.io/ws/v3/YOUR_INFURA_PROJECT_ID`
