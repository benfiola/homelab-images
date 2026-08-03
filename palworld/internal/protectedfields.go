package internal

import (
	"strconv"
	"strings"
)

// protectedField derives one "Server management" ini key from the
// environment, or a fixed constant.
type protectedField struct {
	Key   string
	Value func(env []string) string
}

func lookupEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, prefix); ok {
			return v, true
		}
	}
	return "", false
}

func envOr(name, def string) func(env []string) string {
	return func(env []string) string {
		if v, ok := lookupEnv(env, name); ok && v != "" {
			return v
		}
		return def
	}
}

func quotedEnv(name, def string) func(env []string) string {
	raw := envOr(name, def)
	return func(env []string) string {
		return `"` + raw(env) + `"`
	}
}

func constant(v string) func(env []string) string {
	return func(env []string) string { return v }
}

// protectedFields is the "Server management" allowlist - deployment/infra
// concerns (admin credentials, network config, backup/mod toggles), never
// gameplay tuning.
var protectedFields = []protectedField{
	{"AdminPassword", quotedEnv("ADMIN_PASSWORD", "")},
	{"RCONEnabled", envOr("RCON_ENABLED", "False")},
	{"RCONPort", envOr("RCON_PORT", "25575")},
	// never env-derived - Reboot() depends on the REST API being reachable.
	{"RESTAPIEnabled", constant("True")},
	{"RESTAPIPort", envOr("REST_API_PORT", "8212")},
	{"PublicIP", quotedEnv("PUBLIC_IP", "")},
	{"PublicPort", envOr("PUBLIC_PORT", "8211")},
	{"ServerName", quotedEnv("SERVER_NAME", "Default Palworld Server")},
	{"ServerPassword", quotedEnv("SERVER_PASSWORD", "")},
	{"ServerDescription", quotedEnv("SERVER_DESCRIPTION", "")},
	{"ServerPlayerMaxNum", envOr("PLAYERS", "32")},
	{"CrossplayPlatforms", envOr("CROSSPLAY_PLATFORMS", "(Steam,Xbox,PS5,Mac)")},
	{"ChatPostLimitPerMinute", envOr("CHAT_POST_LIMIT_PER_MINUTE", "30")},
	{"LogFormatType", envOr("LOG_FORMAT_TYPE", "Text")},
	{"bIsUseBackupSaveData", envOr("USE_BACKUP_SAVE_DATA", "True")},
	{"bAllowClientMod", envOr("ALLOW_CLIENT_MOD", "True")},
	{"bIsShowJoinLeftMessage", envOr("IS_SHOW_JOIN_LEFT_MESSAGE", "True")},
	{"bEnableBuildingPlayerUIdDisplay", envOr("ENABLE_BUILDING_PLAYER_UID_DISPLAY", "False")},
	{"Region", quotedEnv("REGION", "")},
	{"bUseAuth", envOr("USE_AUTH", "True")},
	{"BanListURL", quotedEnv("BAN_LIST_URL", "https://b.palworldgame.com/api/banlist.txt")},
}

// ProtectedKeys are the ini keys protectedFields covers, for callers that
// just need to filter them out (editor input, a restored history snapshot).
var ProtectedKeys = func() []string {
	keys := make([]string, len(protectedFields))
	for i, f := range protectedFields {
		keys[i] = f.Key
	}
	return keys
}()

// ReconcileProtectedFields overwrites every protected key in kvs with its
// current env-derived value, adding any that are missing - so "Server
// management" always reflects env vars, never editor input.
func ReconcileProtectedFields(kvs []KV, env []string) []KV {
	for _, f := range protectedFields {
		kvs = Set(kvs, f.Key, f.Value(env))
	}
	return kvs
}

// RestAPIPort resolves the port the REST API is actually running on, for
// the supervisor's own client to target.
func RestAPIPort(env []string) int {
	port, err := strconv.Atoi(envOr("REST_API_PORT", "8212")(env))
	if err != nil {
		return 8212
	}
	return port
}
