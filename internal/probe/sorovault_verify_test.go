package probe

import (
	"testing"

	"github.com/soroworks/soroprobe/internal/abi"
)

func TestVerifySoroVaultLiveWasmHash(t *testing.T) {
	ok, detail := abi.VerifyLiveWasmHash("deadbeef01", "DEADBEEF01")
	if !ok {
		t.Fatalf("expected matching hash to pass, got %s", detail)
	}
	ok, detail = abi.VerifyLiveWasmHash("deadbeef01", "cafebabe99")
	if ok {
		t.Fatalf("expected mismatched hash to warn and disable stale interface, got %s", detail)
	}
}
