package eip712

import (
	"fmt"
	"testing"
)

func TestDecodeAndEncode(t *testing.T) {
	signature := "0xba283b21f7168e53b082ad552d974591abe0f4db5b7032374abbcdcf09e0eadc2c0530ff4ac1d63e19c1ceca2d14b374c86b6c84f46bbd57747b48c21388c4e71c"
	eip712Components := new(EIP712Components)

	// Decode string
	err := eip712Components.Decode(signature)
	if err != nil {
		t.Fatalf("Failed to decode signature: %v", err)
	}

	eip712Components.String()

	// Encode eip712Components back to a string
	encodedSig := eip712Components.Encode()

	if encodedSig != signature {
		t.Fatalf("Expected %s but got %s", signature, encodedSig)
	}

}

func TestDecodeInvalid712Hex(t *testing.T) {
	invalidSignature := "0xinvalidsignature"
	eip712Components := new(EIP712Components)

	err := eip712Components.Decode(invalidSignature)
	if err == nil {
		t.Fatal("Expected error for invalid signature but got none")
	}
}

func TestDecodeEmptySignature(t *testing.T) {
	emptySignature := ""
	eip712Components := new(EIP712Components)

	err := eip712Components.Decode(emptySignature)
	if err == nil {
		t.Fatal("Expected error for empty signature but got none")
	}
}

func TestDecodeInvalidLength(t *testing.T) {
	// Create a hex-encoded signature with an invalid length (not 65 bytes)
	invalidLengthSignature := "0xba283b21f7168e53b082ad552d974591abe0f4db5b7032374abbcdcf09e0eadc" // 64 characters (32 bytes)
	eip712Components := new(EIP712Components)

	err := eip712Components.Decode(invalidLengthSignature)
	if err == nil {
		t.Fatal("Expected error for signature with invalid length but got none")
	}

	expectedErrMsg := fmt.Sprintf("error decoding EIP-712 signature string: invalid length %d bytes (expected %d bytes)", 32, EIP712Length)
	if err.Error() != expectedErrMsg {
		t.Fatalf("Expected error message: '%s' but got: '%s'", expectedErrMsg, err.Error())
	}
}
