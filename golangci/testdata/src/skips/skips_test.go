package skips

import "testing"

func TestExpired(t *testing.T) {
	t.Skip("expires 2020-01-01") // want `the skip expired on 2020-01-01, so fix the test and remove the skip`
}
