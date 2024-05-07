package domain

// TODO: add required fields
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// from MM API: Get users by ids
/*
{
	"id": "string",
	"create_at": 0,
	"update_at": 0,
	"delete_at": 0,
	"username": "string",
	"first_name": "string",
	"last_name": "string",
	"nickname": "string",
	"email": "string",
	"email_verified": true,
	"auth_service": "string",
	"roles": "string",
	"locale": "string",
	"notify_props": {
		"email": "string",
		"push": "string",
		"desktop": "string",
		"desktop_sound": "string",
		"mention_keys": "string",
		"channel": "string",
		"first_name": "string"
		},
	"props": { },
	"last_password_update": 0,
	"last_picture_update": 0,
	"failed_attempts": 0,
	"mfa_active": true,
	"timezone": {
		"useAutomaticTimezone": true,
		"manualTimezone": "string",
		"automaticTimezone": "string"
		},
	"terms_of_service_id": "string",
	"terms_of_service_create_at": 0
}
*/
