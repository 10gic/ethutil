// SPDX-License-Identifier: MIT

// If you want to verify the builtin binary, please use:
// Compiler Version: v0.8.21+commit.d9974bed
// Optimization: 200

pragma solidity ^0.8.21;

struct UserOperation {
    address sender;
    uint256 nonce;
    bytes initCode;
    bytes callData;
    uint256 callGasLimit;
    uint256 verificationGasLimit;
    uint256 preVerificationGas;
    uint256 maxFeePerGas;
    uint256 maxPriorityFeePerGas;
    bytes paymasterAndData;
    bytes signature;
}

contract SimpleAccount {
    address private immutable _entryPoint;
    address _owner;

    uint256 constant internal SIG_VALIDATION_FAILED = 1;

    receive() external payable {}

    constructor(address owner, address entryPoint) {
        _owner = owner;
        _entryPoint = entryPoint;
    }

    /**
     * Validate user's signature and nonce.
     */
    function validateUserOp(UserOperation calldata userOp, bytes32 userOpHash, uint256 missingAccountFunds) external returns (uint256 validationData) {
        require(msg.sender == _entryPoint, "account: not EntryPoint");

        // Step 1: Verify signature
        bytes32 preHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", userOpHash)); // msg before sign
        bytes memory signature = userOp.signature;
        if (signature.length == 65) { // EIP-2098 short signature (64 bytes) is currently not supported
            // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/utils/cryptography/ECDSA.sol
            bytes32 r;
            bytes32 s;
            uint8 v;
            /// @solidity memory-safe-assembly
            assembly {
                r := mload(add(signature, 0x20))
                s := mload(add(signature, 0x40))
                v := byte(0, mload(add(signature, 0x60)))
            }
            if (uint256(s) > 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0) {
                // The `ecrecover` EVM opcode allows for malleable (non-unique) signatures,
                // we rejects them by requiring the `s` value to be in the lower half order
                return SIG_VALIDATION_FAILED;
            }
            address signer = ecrecover(preHash, v, r, s);
            if (signer == address(0)) {
                return SIG_VALIDATION_FAILED;
            }
            if (signer != _owner) {
                return SIG_VALIDATION_FAILED;
            }
        } else {
            return SIG_VALIDATION_FAILED;
        }

        // Step 2: Send to the entrypoint (msg.sender) the missing funds for this transaction
        if (missingAccountFunds != 0) {
            (bool success,) = payable(msg.sender).call{value : missingAccountFunds, gas : type(uint256).max}("");
            (success);
            //ignore failure (its EntryPoint's job to verify, not account.)
        }

        return 0;
    }

    /**
     * execute a transaction (called directly from owner, or by entryPoint)
     */
    function execute(address dest, uint256 value, bytes calldata data) external {
        require(msg.sender == _entryPoint || msg.sender == _owner, "account: not Owner or EntryPoint");

        (bool success, bytes memory result) = dest.call{value : value}(data);
        if (!success) {
            assembly {
                revert(add(result, 32), mload(result))
            }
        }
    }
}

contract SimpleAccountFactory {
    event AccountCreated(address addr);

    constructor() {
    }

    function createAccount(bytes32 _salt, address _accountOwner, address _entryPoint) public returns (address) {
        // This syntax is a newer way to invoke create2 without assembly, you just need to pass salt
        // https://docs.soliditylang.org/en/latest/control-structures.html#salted-contract-creations-create2
        address addr = address(new SimpleAccount{salt: _salt}(_accountOwner, _entryPoint));
        emit AccountCreated(addr);
        return addr;
    }

    function getAddress(bytes32 _salt, address _accountOwner, address _entryPoint) public view returns (address) {
        address deployer = address(this);
        bytes32 bytecodeHash = keccak256(abi.encodePacked(
                type(SimpleAccount).creationCode,
                abi.encode(
                    address(_accountOwner),
                    address(_entryPoint)
                )));
        return computeAddress(_salt, bytecodeHash, deployer);
    }

    function computeAddress(bytes32 salt, bytes32 bytecodeHash, address deployer) internal pure returns (address addr) {
        /// @solidity memory-safe-assembly
        assembly {
            let ptr := mload(0x40) // Get free memory pointer

            // |                   | ↓ ptr ...  ↓ ptr + 0x0B (start) ...  ↓ ptr + 0x20 ...  ↓ ptr + 0x40 ...   |
            // |-------------------|---------------------------------------------------------------------------|
            // | bytecodeHash      |                                                        CCCCCCCCCCCCC...CC |
            // | salt              |                                      BBBBBBBBBBBBB...BB                   |
            // | deployer          | 000000...0000AAAAAAAAAAAAAAAAAAA...AA                                     |
            // | 0xFF              |            FF                                                             |
            // |-------------------|---------------------------------------------------------------------------|
            // | memory            | 000000...00FFAAAAAAAAAAAAAAAAAAA...AABBBBBBBBBBBBB...BBCCCCCCCCCCCCC...CC |
            // | keccak(start, 85) |            ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑ |

            mstore(add(ptr, 0x40), bytecodeHash)
            mstore(add(ptr, 0x20), salt)
            mstore(ptr, deployer) // Right-aligned with 12 preceding garbage bytes
            let start := add(ptr, 0x0b) // The hashed data starts at the final garbage byte which we will set to 0xff
            mstore8(start, 0xff)
            addr := keccak256(start, 85)
        }
    }
}
