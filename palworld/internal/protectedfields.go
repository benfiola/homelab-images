package internal

import "strconv"

// RestAPIPort is the port Palworld's REST API listens on - internal only,
// never an exposed container port, so there's no reason to make it
// configurable.
const RestAPIPort = 8212

func quote(s string) string {
	return `"` + s + `"`
}

// ProtectedKeys are the "Server management" ini keys - deployment/infra
// concerns, never gameplay tuning, never exposed in the editable UI.
var ProtectedKeys = []string{
	"AdminPassword",
	"RCONEnabled",
	"RCONPort",
	"RESTAPIEnabled",
	"RESTAPIPort",
	"PublicIP",
	"PublicPort",
	"ServerName",
	"ServerPassword",
	"ServerDescription",
	"ServerPlayerMaxNum",
	"CrossplayPlatforms",
	"ChatPostLimitPerMinute",
	"LogFormatType",
	"bIsUseBackupSaveData",
	"bAllowClientMod",
	"bIsShowJoinLeftMessage",
	"bEnableBuildingPlayerUIdDisplay",
	"Region",
	"bUseAuth",
	"BanListURL",
}

// ReconcileProtectedFields overwrites every protected key in kvs with its
// deployment-controlled value, adding any that are missing - so "Server
// management" always reflects opts, never editor input. Most of these
// aren't real knobs: in practice nobody tunes them per-deployment, so
// they're fixed at sensible defaults rather than plumbed through as flags.
func ReconcileProtectedFields(kvs []KV, opts Opts) []KV {
	kvs = Set(kvs, "AdminPassword", quote(opts.AdminPassword))
	kvs = Set(kvs, "ServerName", quote(opts.ServerName))
	kvs = Set(kvs, "ServerPassword", quote(opts.ServerPassword))
	kvs = Set(kvs, "ServerPlayerMaxNum", strconv.Itoa(opts.MaxPlayers))

	kvs = Set(kvs, "RCONEnabled", "False")
	kvs = Set(kvs, "RCONPort", "25575")
	// never configurable - Reboot() depends on the REST API being reachable.
	kvs = Set(kvs, "RESTAPIEnabled", "True")
	kvs = Set(kvs, "RESTAPIPort", strconv.Itoa(RestAPIPort))
	kvs = Set(kvs, "PublicIP", `""`)
	kvs = Set(kvs, "PublicPort", strconv.Itoa(opts.Port))
	kvs = Set(kvs, "ServerDescription", `""`)
	kvs = Set(kvs, "CrossplayPlatforms", "(Steam,Xbox,PS5,Mac)")
	kvs = Set(kvs, "ChatPostLimitPerMinute", "30")
	kvs = Set(kvs, "LogFormatType", "Text")
	kvs = Set(kvs, "bIsUseBackupSaveData", "True")
	kvs = Set(kvs, "bAllowClientMod", "True")
	kvs = Set(kvs, "bIsShowJoinLeftMessage", "True")
	kvs = Set(kvs, "bEnableBuildingPlayerUIdDisplay", "False")
	kvs = Set(kvs, "Region", `""`)
	kvs = Set(kvs, "bUseAuth", "True")
	kvs = Set(kvs, "BanListURL", `"https://b.palworldgame.com/api/banlist.txt"`)
	return kvs
}
