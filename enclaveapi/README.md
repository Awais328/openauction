# Enclave API Package

Defines the communication contract between the auction server host and TEE (Trusted Execution Environment) enclaves.

## Types

- **`EnclaveAuctionRequest`** - Request format sent from host to enclave for auction processing
- **`EnclaveAuctionResponse`** - Response format returned from enclave after auction completion
- **`AuctionAttestationDoc`** - Attestation document with cryptographic proofs from secure enclave processing
- **`KeyResponse`** - Response containing public key and attestation from enclave

## Usage

### Host (Exchange) Side
```go
"github.com/cloudx-io/openauction/enclaveapi"

// Send auction to enclave
request := &enclaveapi.EnclaveAuctionRequest{
    Type:      "auction_request",
    AuctionID: "auction-123", // OpenRTB BidRequest.ID
    RoundID:   1,             // Round number within auction
    // ...
}
```

### Enclave Side  
```go
import "github.com/cloudx-io/openauction/enclaveapi"

// Process auction and return response
func processAuction(req enclaveapi.EnclaveAuctionRequest) enclaveapi.EnclaveAuctionResponse {
    // ...
}
```

## Architecture

This package maintains the API contract between two separate binaries:
- **Host**: `auction-server` (web server handling OpenRTB auctions)
- **Enclave**: TEE binary running in AWS Nitro Enclaves

Both packages import from this shared contract to ensure type safety and compatibility.
