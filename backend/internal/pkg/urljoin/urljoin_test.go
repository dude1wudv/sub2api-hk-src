package urljoin

import "testing"

func TestAPIPathPreservesVersionPrefixAndOpaqueQuery(t *testing.T) {
	for _, tc := range []struct{ base, want string }{
		{"https://x", "https://x/v1/messages"},
		{"https://x/v1/", "https://x/v1/messages"},
		{"https://x/proxy/v1?tenant=a%2Fb&x=1#ignored", "https://x/proxy/v1/messages?tenant=a%2Fb&x=1"},
		{"https://x/prefix", "https://x/prefix/v1/messages"},
		{"https://x/v10", "https://x/v10/v1/messages"},
		{"https://x/tenant%2Fv1", "https://x/tenant%2Fv1/v1/messages"},
		{"https://x/prefix/v1/messages", "https://x/prefix/v1/messages"},
	} {
		if got := APIPath(tc.base, "/v1/messages"); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.base, got, tc.want)
		}
	}
	if got := WithQueryParam("https://x/v1/messages?tenant=a%2Fb", "beta", "true"); got != "https://x/v1/messages?tenant=a%2Fb&beta=true" {
		t.Fatal(got)
	}
}
