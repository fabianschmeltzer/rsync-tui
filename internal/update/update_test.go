package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckVerifiesSignedManifest(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, _ := json.Marshal(Manifest{
		Version: "v0.2.0",
		Assets:  map[string]Asset{"linux_amd64": {URL: "https://example.invalid/asset", SHA256: "abcd"}},
	})
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, manifestRaw))

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/releases":
			_ = json.NewEncoder(writer).Encode([]release{{
				TagName: "v0.2.0",
				Assets: []struct {
					Name string `json:"name"`
					URL  string `json:"browser_download_url"`
				}{
					{Name: "manifest.json", URL: server.URL + "/manifest"},
					{Name: "manifest.json.sig", URL: server.URL + "/signature"},
				},
			}})
		case "/manifest":
			_, _ = writer.Write(manifestRaw)
		case "/signature":
			_, _ = writer.Write([]byte(signature))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	available, err := (Client{PublicKey: public, APIURL: server.URL + "/releases"}).Check(context.Background(), "v0.1.0", "beta")
	if err != nil {
		t.Fatal(err)
	}
	if available == nil || available.Version != "v0.2.0" {
		t.Fatalf("unexpected update: %+v", available)
	}
}

func TestCompareVersion(t *testing.T) {
	if compareVersion("v1.0.0", "v1.0.0-beta.1") <= 0 {
		t.Fatal("stable release should sort after prerelease")
	}
	if compareVersion("v0.2.0", "v0.1.9") <= 0 {
		t.Fatal("minor version comparison failed")
	}
}
