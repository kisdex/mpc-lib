This library is a Golang implementation of {2,n}-threshold ECDSA and Ed25519.


This library supports the following functions:

- **2-party ECDSA signature**, using Feldman's VSS generate key shares and Lindell 17 protocol for 2-party
   signature.

- **2-party Ed25519 signature**.

-  **Bip32 key derivation**, support key share unhardened derivation, chaincode is generated by n parties.

- **Key share refresh**, when one party key share is lost or a new participant comes in, support refresh.

See the [Threshold Signature Scheme](docs/Threshold_Signature_Scheme.md) for more detailed information about the
library.

## License

   [Apache-2.0 license](./LICENSE)