package token

import future.keywords

import input.access_token
import input.jwks
import input.method
import input.path

default valid_token := false

valid_token if {
	[valid, _, _] := io.jwt.decode_verify(input.access_token, {"cert": input.jwks, "aud": "account"})
	valid == true
}

default allow := false

allow if {
	regex.match("^/api/organizations", input.path)
	action_is_read
	valid_token
	contains(token_payload.scope, "read:organizations")
}

allow if {
	regex.match("^/api/organizations", input.path)
	action_is_write
	valid_token
	contains(token_payload.scope, "write:organizations")
}

allow if {
	regex.match("^/api/devices", input.path)
	action_is_read
	valid_token
	contains(token_payload.scope, "read:devices")
}

allow if {
	regex.match("^/api/devices", input.path)
	action_is_write
	valid_token
	contains(token_payload.scope, "write:devices")
}

allow if {
	regex.match("^/api/users", input.path)
	action_is_read
	valid_token
	contains(token_payload.scope, "read:users")
}

allow if {
	regex.match("^/api/users", input.path)
	action_is_write
	valid_token
	contains(token_payload.scope, "write:users")
}

allow if {
	regex.match("^/api/fflags", input.path)
	valid_token
}

action_is_read if input.method in ["GET"]

action_is_write := input.method in ["POST", "PATCH", "DELETE"]

token_payload := payload if {
	[_, payload, _] = io.jwt.decode(input.access_token)
}

default user_id = ""

user_id = token_payload.sub

default user_name = ""

user_name = token_payload.preferred_username

default full_name = ""

full_name = token_payload.name