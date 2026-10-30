package pdf

import "testing"

func TestLatin1Bytes(t *testing.T) {
	got, err := latin1Bytes("mât khâu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []byte{0x6d, 0xe2, 0x74, 0x20, 0x6b, 0x68, 0xe2, 0x75}; string(got) != string(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if _, err := latin1Bytes("密码"); err == nil {
		t.Fatalf("expected an error for runes outside Latin-1")
	}
}
