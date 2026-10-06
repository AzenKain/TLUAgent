package constants

const (
	CacheKeyRoleAll                 = "role:all"
	CacheKeyRoleAutoAssignIDs       = "role:auto_assign:ids"
	CacheKeyRoleCountActiveAdminUsers = "role:count:active_admin"
	CacheKeyRoleNamePattern         = "role:name:*"
	CacheKeyRolePermAll             = "role_perm:all"
	CacheKeyPermissionAll           = "permission:all"

	CacheKeyUserAllPattern = "user:*"
	CacheKeyUserSearch     = "user:search:*"
	CacheKeyUserCount      = "user:count:*"

	CacheKeySettingsAll = "settings:all"

	CacheKeyLLMProvidersAll = "llm:providers:all"
	CacheKeyLLMModelsAll    = "llm:models:all"
	CacheKeyLLMChatActive   = "llm:chat:active"

	CacheKeyAgentPromptsAll     = "agent:prompts:all"
	CacheKeyAgentPromptPrefix   = "agent:prompt:"
	CacheKeyAgentSkillsAll      = "agent:skills:all"
	CacheKeyAgentSkillsEnabled  = "agent:skills:enabled"
	CacheKeyAgentSkillPrefix    = "agent:skill:"
)
