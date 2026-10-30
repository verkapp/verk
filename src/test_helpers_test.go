package src

import "testing"

func newTestStore(t *testing.T) *store {
	t.Helper()
	st, err := openStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("openStoreAt: %v", err)
	}
	t.Cleanup(func() {
		_ = st.close()
	})
	return st
}
