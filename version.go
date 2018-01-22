package flagstate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

var (
	GitVersion  = ""
	BuildTime   = ""
	BuildString = ""
	BuildId     = ""
)

func init() {
	if GitVersion == "" && BuildTime == "" {
		BuildString = "<no build information supplied>"
		BuildId = "XXXXXXXXXX"
	} else {
		var hashMaterial string
		if strings.HasSuffix(GitVersion, "*") {
			hashMaterial = GitVersion + "|" + BuildTime
		} else {
			hashMaterial = GitVersion
		}

		hash := sha256.New()
		hash.Write([]byte(hashMaterial))
		hex := hex.EncodeToString(hash.Sum([]byte{}))
		BuildId = hex[0:10] // Truncate to 10 digits for compactness

		BuildString = fmt.Sprintf("GitVersion=%s, BuildTime=%s, BuildId=%s", GitVersion, BuildTime, BuildId)
	}
}
